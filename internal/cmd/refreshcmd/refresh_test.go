package refreshcmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/open-cli-collective/cli-common/statedirtest"

	"github.com/open-cli-collective/google-readonly/internal/cache"
	"github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

type stubLister struct {
	drives    []*drive.SharedDrive
	err       error
	callCount int
}

func (s *stubLister) ListSharedDrives(_ context.Context, _ int64) ([]*drive.SharedDrive, error) {
	s.callCount++
	return s.drives, s.err
}

// panickingFactory makes any client construction a hard failure. Used to
// pin the invariant that --status does not invoke the factory.
type panickingFactory struct {
	callCount int
}

func (p *panickingFactory) factory(_ context.Context) (DriveLister, error) {
	p.callCount++
	panic("client factory must not be invoked")
}

func TestRefresh_StatusDoesNotInvokeClientFactory(t *testing.T) {
	statedirtest.Hermetic(t)

	pf := &panickingFactory{}
	cmd := newCommandWithDeps(pf.factory)
	cmd.SetArgs([]string{"--status"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("--status returned error: %v", err)
	}
	testutil.Equal(t, pf.callCount, 0)
	testutil.Contains(t, out.String(), "RESOURCE | FETCHED_AT | AGE | TTL | STATUS")
	testutil.Contains(t, out.String(), "uninitialized")
}

func TestRefresh_DoesInvokeClientFactory(t *testing.T) {
	statedirtest.Hermetic(t)

	stub := &stubLister{drives: []*drive.SharedDrive{{ID: "0A1", Name: "Eng"}}}
	factoryCalls := 0
	factory := func(_ context.Context) (DriveLister, error) {
		factoryCalls++
		return stub, nil
	}

	cmd := newCommandWithDeps(factory)
	cmd.SetArgs([]string{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("refresh returned error: %v", err)
	}
	testutil.Equal(t, factoryCalls, 1)
	testutil.Equal(t, stub.callCount, 1)
	testutil.Contains(t, out.String(), "Refreshing drives... 1 entries")
}

func TestRefresh_Status_Fresh(t *testing.T) {
	statedirtest.Hermetic(t)

	// Prime the cache with a fresh write.
	c, err := cache.New()
	testutil.NoError(t, err)
	testutil.NoError(t, c.SetDrives([]*cache.CachedDrive{{ID: "0A1", Name: "Eng"}}))

	cmd := newCommandWithDeps((&panickingFactory{}).factory)
	cmd.SetArgs([]string{"--status"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--status returned error: %v", err)
	}
	got := out.String()
	testutil.Contains(t, got, "drives | ")
	testutil.Contains(t, got, " | fresh")
}

func TestRefresh_Status_Stale(t *testing.T) {
	statedirtest.Hermetic(t)

	c, err := cache.New()
	testutil.NoError(t, err)
	testutil.NoError(t, c.SetDrives([]*cache.CachedDrive{{ID: "0A1", Name: "Eng"}}))

	// Advance gro's cache clock past the drives TTL (24h).
	origNow := cache.NowFnForTest()
	cache.SetNowFnForTest(func() time.Time { return time.Now().Add(48 * time.Hour) })
	defer cache.SetNowFnForTest(origNow)

	cmd := newCommandWithDeps((&panickingFactory{}).factory)
	cmd.SetArgs([]string{"--status"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--status returned error: %v", err)
	}
	testutil.Contains(t, out.String(), " | stale")
}

func TestRefresh_PositionalArgs(t *testing.T) {
	statedirtest.Hermetic(t)

	stub := &stubLister{drives: []*drive.SharedDrive{{ID: "0A1", Name: "Eng"}}}
	cmd := newCommandWithDeps(func(_ context.Context) (DriveLister, error) { return stub, nil })
	cmd.SetArgs([]string{"drives"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("refresh drives returned error: %v", err)
	}
	testutil.Equal(t, stub.callCount, 1)
}

func TestRefresh_UnknownResource(t *testing.T) {
	statedirtest.Hermetic(t)

	cmd := newCommandWithDeps((&panickingFactory{}).factory)
	cmd.SetArgs([]string{"unknown"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown resource, got nil")
	}
	testutil.Contains(t, err.Error(), "unknown")
}

func TestRefresh_PropagatesListError(t *testing.T) {
	statedirtest.Hermetic(t)

	stub := &stubLister{err: errors.New("boom")}
	cmd := newCommandWithDeps(func(_ context.Context) (DriveLister, error) { return stub, nil })
	cmd.SetArgs([]string{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from list failure, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected error to wrap 'boom', got %q", err.Error())
	}
}

func TestRefresh_FactoryConstructionError(t *testing.T) {
	statedirtest.Hermetic(t)

	factoryErr := errors.New("auth required")
	cmd := newCommandWithDeps(func(_ context.Context) (DriveLister, error) { return nil, factoryErr })
	cmd.SetArgs([]string{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from factory failure, got nil")
	}
	if !errors.Is(err, factoryErr) {
		t.Fatalf("expected error to wrap factoryErr, got %q", err.Error())
	}
}
