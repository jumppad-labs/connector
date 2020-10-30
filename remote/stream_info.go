package remote

import (
	"context"
	"sync"

	"github.com/shipyard-run/connector/protos/shipyard"
)

type streams []*streamInfo

var streamMutex sync.Mutex

func (c *streams) add(si *streamInfo) {
	streamMutex.Lock()
	defer streamMutex.Unlock()

	*c = append(*c, si)
}

func (c *streams) remove(si *streamInfo) {
	streamMutex.Lock()
	defer streamMutex.Unlock()

	newSlice := streams{}
	for _, s := range *c {
		if s != si {
			newSlice = append(newSlice, si)
		}
	}

	*c = newSlice
}

func (c *streams) findByRemoteAddr(addr string) (*streamInfo, bool) {
	for _, v := range *c {
		if v.addr == addr {
			return v, true
		}
	}

	return nil, false
}

func (c *streams) findByServiceID(id string) (*streamInfo, bool) {
	for _, v := range *c {
		found := false
		v.services.iterate(func(svcid string, svc *service) bool {
			if id == svcid {
				found = true
				return false // stop itterating
			}

			return true
		})

		if found {
			return v, true
		}
	}

	return nil, false
}

func (c *streams) findByRemoteConnection(rc interface{}) (*streamInfo, bool) {
	for _, v := range *c {
		if rc == v.grpcConn.conn {
			return v, true
		}
	}

	return nil, false
}

type streamInfo struct {
	connecting  bool
	addr        string
	grpcConn    *grpcConn
	services    *services
	updateMutex sync.Mutex
}

// returns a grpc connection in a thread safe way
func (si *streamInfo) closeGRPCConn() {
	si.updateMutex.Lock()
	defer si.updateMutex.Unlock()

	if si.grpcConn != nil {
		si.grpcConn.Close()
	}
}

func (si *streamInfo) setGRPCConn(g *grpcConn) {
	si.updateMutex.Lock()
	defer si.updateMutex.Unlock()

	si.grpcConn = g
}

func (si *streamInfo) isConnecting() bool {
	si.updateMutex.Lock()
	defer si.updateMutex.Unlock()

	return si.connecting
}

func (si *streamInfo) setConnecting(value bool) {

	si.updateMutex.Lock()
	defer si.updateMutex.Unlock()

	si.connecting = value
}

func newStreamInfo() *streamInfo {
	return &streamInfo{
		services: newServices(),
	}
}

type grpcConn struct {
	ctx       context.Context
	cancel    context.CancelFunc
	conn      interface{}
	Closed    bool
	syncMutex sync.Mutex
}

func newGRPCConn(c interface{}) *grpcConn {
	ctx, cf := context.WithCancel(context.Background())
	return &grpcConn{
		ctx:       ctx,
		conn:      c,
		cancel:    cf,
		syncMutex: sync.Mutex{},
	}
}

func (r *grpcConn) Send(data *shipyard.OpenData) {
	r.syncMutex.Lock()
	defer r.syncMutex.Unlock()

	// do nothing if closed
	if r.Closed {
		return
	}

	switch c := r.conn.(type) {
	case shipyard.RemoteConnection_OpenStreamClient:
		c.Send(data)
	case shipyard.RemoteConnection_OpenStreamServer:
		c.Send(data)
	}
}

func (r *grpcConn) Recv() (*shipyard.OpenData, error) {
	switch c := r.conn.(type) {
	case shipyard.RemoteConnection_OpenStreamClient:
		return c.Recv()
	case shipyard.RemoteConnection_OpenStreamServer:
		return c.Recv()
	}

	return nil, nil
}

func (r *grpcConn) Close() {
	r.syncMutex.Lock()
	defer r.syncMutex.Unlock()

	switch c := r.conn.(type) {
	case shipyard.RemoteConnection_OpenStreamClient:
		c.CloseSend()
	}

	// cancel the context
	r.Closed = true
	r.cancel()
}

// wrap the context cancelled
func (r *grpcConn) Done() <-chan struct{} {
	return r.ctx.Done()
}
