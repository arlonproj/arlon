package gitrepo

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

const content = `
		{
			"message": "This is some content which doesn't have to necessarily be JSON"
		}
	`

const (
	expectedDefaultRepoUrl = "https://github.com/exampleuser/examplerepo"
	expectedReposLen       = 2
)

var (
	expectedAliases = []string{"default", "notDefault"}
	expectedUrls    = []string{expectedDefaultRepoUrl, "https://github.com/exampleuser/secondrepo"}
)

const contentTpl = `
	{
		"default": {
			"url": "%s",
			"alias": "%s"
		},
		"repos": [
			{
				"url": "%s",
				"alias": "%s"
			},
			{
				"url": "%s",
				"alias": "%s"
			}
		]
	}
	`

var cfg = fmt.Sprintf(contentTpl, expectedDefaultRepoUrl, RepoDefaultCtx, expectedDefaultRepoUrl, RepoDefaultCtx, expectedUrls[1], expectedAliases[1])

func TestLoadRepoCfg(t *testing.T) {
	reader := bytes.NewReader([]byte(cfg))
	ctxCfg, err := LoadRepoCfg(reader)
	require.NoError(t, err)
	require.Equal(t, expectedDefaultRepoUrl, ctxCfg.Default.Url)
	require.Equal(t, RepoDefaultCtx, ctxCfg.Default.Alias)
	require.Equal(t, expectedReposLen, len(ctxCfg.Repos))
	require.Equal(t, expectedDefaultRepoUrl, ctxCfg.Repos[0].Url)
	require.Equal(t, RepoDefaultCtx, ctxCfg.Repos[0].Alias)
	require.Equal(t, expectedUrls[1], ctxCfg.Repos[1].Url)
	require.Equal(t, expectedAliases[1], ctxCfg.Repos[1].Alias)
}

func TestAliasExists(t *testing.T) {
	repoList := []RepoCtx{
		{
			Url:   "https://github.com/someuser/somerepo",
			Alias: RepoDefaultCtx,
		},
		{
			Url:   "https://github.com/anotheruser/another_repo",
			Alias: "another",
		},
		{
			Url:   "https://github.com/random/randomrepo",
			Alias: "randomRepo",
		},
	}
	testCases := []struct {
		Desc        string
		RepoList    []RepoCtx
		TargetAlias string
		Exists      bool
	}{
		{RepoList: repoList, TargetAlias: RepoDefaultCtx, Exists: true, Desc: "An alias that exists"},
		{RepoList: repoList, TargetAlias: "not-found", Exists: false, Desc: "An non existent alias"},
		{RepoList: []RepoCtx{}, TargetAlias: "something", Exists: false, Desc: "An empty slice"},
		{RepoList: nil, TargetAlias: "something", Exists: false, Desc: "A nil slice"},
	}

	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {
			exists := AliasExists(tc.RepoList, tc.TargetAlias)
			require.Equal(t, tc.Exists, exists)
		})
	}
}

func TestTruncateFile(t *testing.T) {
	const fileName = "toBeTruncated"
	f, err := os.CreateTemp("", fileName)
	t.Cleanup(func() {
		_ = os.Remove(f.Name())
	})
	require.NoError(t, err)
	bytesWritten, err := f.Write([]byte(content))
	require.Equal(t, len(content), bytesWritten)
	require.NoError(t, err)
	err = TruncateFile(f)
	require.NoError(t, err)
	info, err := f.Stat()
	require.NoError(t, err)
	require.Equal(t, int64(0), info.Size())
}

func TestStoreRepoCfg(t *testing.T) {
	var buff []byte
	var writeBuffer = bytes.NewBuffer(buff)
	err := StoreRepoCfg(writeBuffer, []byte(content))
	require.NoError(t, err)
	require.Equal(t, writeBuffer.Len(), len(content))
	readContent, err := io.ReadAll(writeBuffer)
	require.NoError(t, err)
	require.Equal(t, []byte(content), readContent)
}

func TestGetAlias(t *testing.T) {
	testCases := []struct {
		Desc        string
		TargetAlias string
		Err         error
		Reader      io.Reader
		RepoCtxObj  *RepoCtx
	}{
		{
			Desc:        "Existing Alias",
			TargetAlias: expectedAliases[0],
			Err:         nil,
			Reader:      bytes.NewReader([]byte(cfg)),
			RepoCtxObj: &RepoCtx{
				Url:   expectedUrls[0],
				Alias: expectedAliases[0],
			},
		},
		{
			Desc:        "Existing Alias",
			TargetAlias: expectedAliases[1],
			Err:         nil,
			Reader:      bytes.NewReader([]byte(cfg)),
			RepoCtxObj: &RepoCtx{
				Url:   expectedUrls[1],
				Alias: expectedAliases[1],
			},
		},
		{
			Desc:        "Non Existing alias",
			TargetAlias: "bogus",
			Err:         ErrNotFound,
			Reader:      bytes.NewReader([]byte(cfg)),
			RepoCtxObj:  nil,
		},
		{
			Desc:        "Nil slice for repos",
			TargetAlias: "doesnot-matter",
			Err:         ErrNotFound,
			Reader: bytes.NewReader([]byte(`{
				"default": {}
			}`)),
			RepoCtxObj: nil,
		},
		{
			Desc:        "Empty Slice for repos",
			TargetAlias: "doesnot-matter",
			Err:         ErrNotFound,
			Reader: bytes.NewReader([]byte(`{
				"default": {},
				"repos": []
			}`)),
			RepoCtxObj: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {
			repoCtx, err := getAlias(tc.TargetAlias, tc.Reader)
			require.Equal(t, tc.Err, err)
			require.EqualValues(t, tc.RepoCtxObj, repoCtx)
		})
	}
}
