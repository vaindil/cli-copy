package list

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/browser"
	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewCmdList(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		isTTY   bool
		want    ListOptions
		wantErr string
	}{
		{
			name:  "no arguments",
			args:  "",
			isTTY: true,
			want: ListOptions{
				Limit:        30,
				WebMode:      false,
				Organization: "",
			},
		},
		{
			name:  "limit",
			args:  "--limit 1",
			isTTY: true,
			want: ListOptions{
				Limit:        1,
				WebMode:      false,
				Organization: "",
			},
		},
		{
			name:  "org",
			args:  "--org \"my-org\"",
			isTTY: true,
			want: ListOptions{
				Limit:        30,
				WebMode:      false,
				Organization: "my-org",
			},
		},
		{
			name:  "web mode",
			args:  "--web",
			isTTY: true,
			want: ListOptions{
				Limit:        30,
				WebMode:      true,
				Organization: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			ios.SetStdoutTTY(tt.isTTY)
			ios.SetStdinTTY(tt.isTTY)
			ios.SetStderrTTY(tt.isTTY)

			f := &cmdutil.Factory{
				IOStreams: ios,
			}

			var opts *ListOptions
			cmd := NewCmdList(f, func(o *ListOptions) error {
				opts = o
				return nil
			})
			cmd.PersistentFlags().StringP("repo", "R", "", "")

			argv, err := shlex.Split(tt.args)
			require.NoError(t, err)
			cmd.SetArgs(argv)

			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err = cmd.ExecuteC()
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want.Limit, opts.Limit)
			assert.Equal(t, tt.want.WebMode, opts.WebMode)
			assert.Equal(t, tt.want.Organization, opts.Organization)
		})
	}
}

func Test_listRun(t *testing.T) {
	tests := []struct {
		name       string
		isTTY      bool
		opts       ListOptions
		wantErr    string
		wantStdout string
		wantStderr string
	}{
		{
			name:  "list repo rulesets",
			isTTY: true,
			wantStdout: heredoc.Doc(`

				Showing 3 of 3 rulesets in OWNER/REPO

				ID  NAME    STATUS    TARGET
				4   test    evaluate  branch
				42  asdf    active    branch
				77  foobar  disabled  branch
			`),
			wantStderr: "",
		},
		{
			name:  "list org rulesets",
			isTTY: true,
			opts: ListOptions{
				Organization: "my-org",
			},
			wantStdout: heredoc.Doc(`

				Showing 3 of 3 rulesets in my-org

				ID  NAME    STATUS    TARGET
				4   test    evaluate  branch
				42  asdf    active    branch
				77  foobar  disabled  branch
			`),
			wantStderr: "",
		},
		{
			name:  "machine-readable",
			isTTY: false,
			wantStdout: heredoc.Doc(`
				4	test	evaluate	branch
				42	asdf	active	branch
				77	foobar	disabled	branch
			`),
			wantStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.isTTY)
			ios.SetStdinTTY(tt.isTTY)
			ios.SetStderrTTY(tt.isTTY)

			fakeHTTP := &httpmock.Registry{}
			fakeHTTP.Register(httpmock.GraphQL(`query RulesetList\b`), httpmock.FileResponse("./fixtures/rulesetList.json"))

			tt.opts.IO = ios
			tt.opts.HttpClient = func() (*http.Client, error) {
				return &http.Client{Transport: fakeHTTP}, nil
			}
			tt.opts.BaseRepo = func() (ghrepo.Interface, error) {
				return ghrepo.FromFullName("OWNER/REPO")
			}
			tt.opts.Browser = &browser.Stub{}
			tt.opts.Config = func() (config.Config, error) {
				return config.NewBlankConfig(), nil
			}

			err := listRun(&tt.opts)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantStdout, stdout.String())
			assert.Equal(t, tt.wantStderr, stderr.String())
		})
	}
}