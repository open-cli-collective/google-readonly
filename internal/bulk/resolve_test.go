package bulk

import (
	"fmt"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestResolveIDs_Args(t *testing.T) {
	t.Parallel()
	ids, err := ResolveIDs(Config{Args: []string{"id1", "id2"}}, nil)
	testutil.NoError(t, err)
	testutil.Len(t, ids, 2)
	testutil.Equal(t, ids[0], "id1")
	testutil.Equal(t, ids[1], "id2")
}

func TestResolveIDs_Query(t *testing.T) {
	t.Parallel()
	queryFn := func(q string) ([]string, error) {
		testutil.Equal(t, q, "is:unread")
		return []string{"msg1", "msg2", "msg3"}, nil
	}

	ids, err := ResolveIDs(Config{Query: "is:unread"}, queryFn)
	testutil.NoError(t, err)
	testutil.Len(t, ids, 3)
}

func TestResolveIDs_QueryError(t *testing.T) {
	t.Parallel()
	queryFn := func(_ string) ([]string, error) {
		return nil, fmt.Errorf("search failed")
	}

	_, err := ResolveIDs(Config{Query: "is:unread"}, queryFn)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "search failed")
}

func TestResolveIDs_NoSources(t *testing.T) {
	t.Parallel()
	_, err := ResolveIDs(Config{}, nil)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "provide message IDs")
}

func TestResolveIDs_MultipleSources(t *testing.T) {
	t.Parallel()
	_, err := ResolveIDs(Config{Args: []string{"id1"}, Query: "test"}, nil)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "only one input source")
}

func TestResolveIDs_ArgsAndStdin(t *testing.T) {
	t.Parallel()
	_, err := ResolveIDs(Config{Args: []string{"id1"}, Stdin: true}, nil)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "only one input source")
}

func TestResolveIDs_AllThree(t *testing.T) {
	t.Parallel()
	_, err := ResolveIDs(Config{Args: []string{"id1"}, Stdin: true, Query: "test"}, nil)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "only one input source")
}
