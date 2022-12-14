package http

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/gososy/sorpc/registry"
	"github.com/gososy/sorpc/selector"
)

func TestParseTarget(t *testing.T) {
	target, err := parseTarget("localhost:8000", true)
	if err != nil {
		t.Errorf("expect %v, got %v", nil, err)
	}
	if !reflect.DeepEqual(&Target{Scheme: "http", Authority: "localhost:8000"}, target) {
		t.Errorf("expect %v, got %v", &Target{Scheme: "http", Authority: "localhost:8000"}, target)
	}

	target, err = parseTarget("discovery:///demo", true)
	if err != nil {
		t.Errorf("expect %v, got %v", nil, err)
	}
	if !reflect.DeepEqual(&Target{Scheme: "discovery", Authority: "", Endpoint: "demo"}, target) {
		t.Errorf("expect %v, got %v", &Target{Scheme: "discovery", Authority: "", Endpoint: "demo"}, target)
	}

	target, err = parseTarget("127.0.0.1:8000", true)
	if err != nil {
		t.Errorf("expect %v, got %v", nil, err)
	}
	if !reflect.DeepEqual(&Target{Scheme: "http", Authority: "127.0.0.1:8000"}, target) {
		t.Errorf("expect %v, got %v", &Target{Scheme: "http", Authority: "127.0.0.1:8000"}, target)
	}

	target, err = parseTarget("https://127.0.0.1:8000", false)
	if err != nil {
		t.Errorf("expect %v, got %v", nil, err)
	}
	if !reflect.DeepEqual(&Target{Scheme: "https", Authority: "127.0.0.1:8000"}, target) {
		t.Errorf("expect %v, got %v", &Target{Scheme: "https", Authority: "127.0.0.1:8000"}, target)
	}

	target, err = parseTarget("127.0.0.1:8000", false)
	if err != nil {
		t.Errorf("expect %v, got %v", nil, err)
	}
	if !reflect.DeepEqual(&Target{Scheme: "https", Authority: "127.0.0.1:8000"}, target) {
		t.Errorf("expect %v, got %v", &Target{Scheme: "https", Authority: "127.0.0.1:8000"}, target)
	}
}

type mockRebalancer struct{}

func (m *mockRebalancer) Apply(nodes []selector.Node) {}

type mockDiscoveries struct {
	isSecure bool
	nextErr  bool
	stopErr  bool
}

func (d *mockDiscoveries) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	return nil, nil
}

const errServiceName = "needErr"

func (d *mockDiscoveries) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	if serviceName == errServiceName {
		return nil, errors.New("mock test service name watch err")
	}
	return &mockWatch{ctx: ctx, isSecure: d.isSecure, nextErr: d.nextErr, stopErr: d.stopErr}, nil
}

type mockWatch struct {
	ctx context.Context

	isSecure bool
	count    int

	nextErr bool
	stopErr bool
}

func (m *mockWatch) Next() ([]*registry.ServiceInstance, error) {
	select {
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	default:
	}
	if m.nextErr {
		return nil, errors.New("mock test error")
	}
	if m.count == 1 {
		return nil, errors.New("mock test error")
	}
	m.count++
	instance := &registry.ServiceInstance{
		ID:        "1",
		Name:      "kratos",
		Version:   "v1",
		Metadata:  map[string]string{},
		Endpoints: []string{fmt.Sprintf("http://127.0.0.1:9001?isSecure=%s", strconv.FormatBool(m.isSecure))},
	}
	if m.count > 3 {
		time.Sleep(time.Millisecond * 500)
	}
	return []*registry.ServiceInstance{instance}, nil
}

func (m *mockWatch) Stop() error {
	if m.stopErr {
		return errors.New("mock test error")
	}
	// ?????? next ????????????
	m.nextErr = true
	return nil
}

func TestResolver(t *testing.T) {
	ta, err := parseTarget("discovery://helloworld", true)
	if err != nil {
		t.Errorf("parse err %v", err)
		return
	}

	// ?????? ????????????
	_, err = newResolver(context.Background(), &mockDiscoveries{true, false, false}, ta, &mockRebalancer{}, false, false)
	if err != nil {
		t.Errorf("expect %v, got %v", nil, err)
	}

	// ?????? ??????????????????
	_, err = newResolver(context.Background(), &mockDiscoveries{false, false, false}, ta, &mockRebalancer{}, true, true)
	if err != nil {
		t.Errorf("expect %v, got %v", nil, err)
	}

	// ?????? ?????? next ?????? ?????? stop ??????
	_, err = newResolver(context.Background(), &mockDiscoveries{false, true, true}, ta, &mockRebalancer{}, true, true)
	if err == nil {
		t.Errorf("expect err, got nil")
	}

	// ?????? service name watch ??????
	_, err = newResolver(context.Background(), &mockDiscoveries{false, true, true}, &Target{
		Scheme:   "discovery",
		Endpoint: errServiceName,
	}, &mockRebalancer{}, true, true)
	if err == nil {
		t.Errorf("expect err, got nil")
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// ???????????????????????? context.Canceled
	r, err := newResolver(cancelCtx, &mockDiscoveries{false, false, false}, ta, &mockRebalancer{}, false, false)
	if err != nil {
		t.Errorf("expect %v, got %v", nil, err)
	}
	_ = r.Close()

	// ?????? ???????????????????????????????????????
	_, err = newResolver(cancelCtx, &mockDiscoveries{false, false, true}, ta, &mockRebalancer{}, true, true)
	if err == nil {
		t.Errorf("expect ctx cancel err, got nil")
	}
}
