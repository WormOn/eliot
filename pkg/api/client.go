package api

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/ernoaapa/eliot/pkg/api/mapping"
	containers "github.com/ernoaapa/eliot/pkg/api/services/containers/v1"
	device "github.com/ernoaapa/eliot/pkg/api/services/device/v1"
	pods "github.com/ernoaapa/eliot/pkg/api/services/pods/v1"
	"github.com/ernoaapa/eliot/pkg/api/stream"
	"github.com/ernoaapa/eliot/pkg/config"
	"github.com/ernoaapa/eliot/pkg/progress"
	"github.com/rs/xid"
)

// Client connects directly to device RPC API
type Client struct {
	Namespace string
	Endpoint  config.Endpoint
	ctx       context.Context
}

// NewClient creates new RPC server client
func NewClient(namespace string, endpoint config.Endpoint) *Client {
	return &Client{
		namespace,
		endpoint,
		context.Background(),
	}
}

// GetInfo calls server and get device info
func (c *Client) GetInfo() (*device.Info, error) {
	conn, err := grpc.Dial(c.Endpoint.URL, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := device.NewDeviceClient(conn)
	resp, err := client.Info(c.ctx, &device.InfoRequest{})
	if err != nil {
		return nil, err
	}

	return resp.GetInfo(), nil
}

// GetPods calls server and fetches all pods information
func (c *Client) GetPods() ([]*pods.Pod, error) {
	conn, err := grpc.Dial(c.Endpoint.URL, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pods.NewPodsClient(conn)
	resp, err := client.List(c.ctx, &pods.ListPodsRequest{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	return resp.GetPods(), nil
}

// GetPod return Pod by name
func (c *Client) GetPod(podName string) (*pods.Pod, error) {
	pods, err := c.GetPods()
	if err != nil {
		return nil, err
	}

	for _, pod := range pods {
		if pod.Metadata.Name == podName {
			return pod, nil
		}
	}
	return nil, fmt.Errorf("Pod with name [%s] not found", podName)
}

// CreatePod creates new pod to the device
func (c *Client) CreatePod(status chan<- []*progress.ImageFetch, pod *pods.Pod, opts ...PodOpts) error {
	for _, o := range opts {
		err := o(pod)
		if err != nil {
			return err
		}
	}

	conn, err := grpc.Dial(c.Endpoint.URL, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pods.NewPodsClient(conn)
	stream, err := client.Create(c.ctx, &pods.CreatePodRequest{
		Pod: pod,
	})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			err = stream.CloseSend()
			return err
		}
		if err != nil {
			return err
		}

		status <- mapping.MapAPIModelToImageFetchProgress(resp.Images)
	}
}

// StartPod starts created pod in device
func (c *Client) StartPod(name string) (*pods.Pod, error) {
	conn, err := grpc.Dial(c.Endpoint.URL, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pods.NewPodsClient(conn)
	resp, err := client.Start(c.ctx, &pods.StartPodRequest{
		Namespace: c.Namespace,
		Name:      name,
	})
	if err != nil {
		return nil, err
	}

	return resp.GetPod(), nil
}

// DeletePod removes pod from the device
func (c *Client) DeletePod(pod *pods.Pod) (*pods.Pod, error) {
	conn, err := grpc.Dial(c.Endpoint.URL, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pods.NewPodsClient(conn)

	resp, err := client.Delete(c.ctx, &pods.DeletePodRequest{
		Namespace: pod.Metadata.Namespace,
		Name:      pod.Metadata.Name,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetPod(), nil
}

// Attach hooks to container main process stdin/stout
func (c *Client) Attach(containerID string, attachIO AttachIO, hooks ...AttachHooks) (err error) {
	done := make(chan struct{})
	errc := make(chan error)

	md := metadata.Pairs(
		"namespace", c.Namespace,
		"container", containerID,
	)
	ctx, cancel := context.WithCancel(metadata.NewOutgoingContext(c.ctx, md))
	defer cancel()

	conn, err := grpc.Dial(c.Endpoint.URL, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := containers.NewContainersClient(conn)
	log.Debugf("Open connection to server to start stdin/stdout streaming")
	s, err := client.Attach(ctx)
	if err != nil {
		return err
	}

	go func() {
		errc <- stream.PipeStdout(s, attachIO.Stdout, attachIO.Stderr)
	}()

	if attachIO.Stdin != nil {
		go func() {
			errc <- stream.PipeStdin(s, attachIO.Stdin)
		}()
	}

	for _, hook := range hooks {
		go hook(c.Endpoint, done)
	}

	for {
		err := <-errc
		close(done)
		return err
	}
}

// Exec executes command inside some container
func (c *Client) Exec(containerID string, args []string, tty bool, attachIO AttachIO, hooks ...AttachHooks) (err error) {
	done := make(chan struct{})
	errc := make(chan error)

	md := metadata.Pairs(
		"namespace", c.Namespace,
		"container", containerID,
		"execid", xid.New().String(),
		"args", strings.Join(args, " "),
		"tty", strconv.FormatBool(tty),
	)
	ctx, cancel := context.WithCancel(metadata.NewOutgoingContext(c.ctx, md))
	defer cancel()

	conn, err := grpc.Dial(c.Endpoint.URL, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := containers.NewContainersClient(conn)
	log.Debugf("Open connection to server to start stdin/stdout streaming")
	s, err := client.Exec(ctx)
	if err != nil {
		return err
	}

	go func() {
		errc <- stream.PipeStdout(s, attachIO.Stdout, attachIO.Stderr)
	}()

	if attachIO.Stdin != nil {
		go func() {
			errc <- stream.PipeStdin(s, attachIO.Stdin)
		}()
	}

	for _, hook := range hooks {
		go hook(c.Endpoint, done)
	}

	for {
		err := <-errc
		close(done)
		return err
	}
}

// Signal sends kill signal to container process
func (c *Client) Signal(containerID string, signal syscall.Signal) (err error) {
	conn, err := grpc.Dial(c.Endpoint.URL, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := containers.NewContainersClient(conn)

	_, err = client.Signal(c.ctx, &containers.SignalRequest{
		Namespace:   c.Namespace,
		ContainerID: containerID,
		Signal:      int32(signal),
	})

	return err
}
