package git

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCommand struct {
	mock.Mock
}

func (m *mockCommand) CombinedOutput() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockCommand) SetDir(dir string) {
	m.Called(dir)
}

func (m *mockCommand) GetArgs() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

type mockedExecCommanderOptions struct {
	combinedOutputOut []byte
	combinedOutputErr error
	setDirCalledWith  string
	getArgsOut        []string
	onExecCommander   func(name string, args ...string)
}

func newMockedExecCommander(options mockedExecCommanderOptions) (*mockCommand, func()) {
	if options.setDirCalledWith == "" {
		options.setDirCalledWith = "dir"
	}
	if options.combinedOutputOut == nil {
		options.combinedOutputOut = []byte("mock output")
	}
	if options.getArgsOut == nil {
		options.getArgsOut = []string{"arg2", "arg3"}
	}

	mockCmd := new(mockCommand)
	mockCmd.On("SetDir", options.setDirCalledWith).Return()
	mockCmd.On("CombinedOutput").Return(options.combinedOutputOut, options.combinedOutputErr)
	mockCmd.On("GetArgs").Return(options.getArgsOut)

	orig := execCommander
	execCommander = func(name string, args ...string) Command {
		if options.onExecCommander != nil {
			options.onExecCommander(name, args...)
		}
		return mockCmd
	}

	return mockCmd, func() { execCommander = orig }
}

func TestBuildGenericArgs(t *testing.T) {
	tests := []struct {
		name   string
		base   []string
		opts   DiffOptions
		expect []string
	}{
		{
			name:   "Diff staged with unified",
			base:   []string{"diff", "--cached"},
			opts:   DiffOptions{Unified: 5},
			expect: []string{"diff", "--cached", "--unified=5"},
		},
		{
			name:   "Diff refs with FindRenames enabled",
			base:   []string{"diff", "dev", "main"},
			opts:   DiffOptions{Unified: 1, FindRenames: true},
			expect: []string{"diff", "dev", "main", "--unified=1", "--find-renames"},
		},
		{
			name:   "Diff commit With filters",
			base:   []string{"show", "8062f1"},
			opts:   DiffOptions{Unified: 2, Filters: []string{"a.txt", "b.txt"}},
			expect: []string{"show", "8062f1", "--unified=2", "--", "a.txt", "b.txt"},
		},
		{
			name:   "All together",
			base:   []string{"diff"},
			opts:   DiffOptions{Unified: 7, FindRenames: true, Filters: []string{"f.go"}},
			expect: []string{"diff", "--unified=7", "--find-renames", "--", "f.go"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildGenericArgs(tt.base, tt.opts)
			assert.Equal(t, tt.expect, args)
		})
	}
}

func TestDiffRefs(t *testing.T) {
	var execCommanderCalledWith map[string]any
	mockCmd, resetExecCommander := newMockedExecCommander(mockedExecCommanderOptions{
		setDirCalledWith: ".",
		onExecCommander: func(name string, args ...string) {
			execCommanderCalledWith = map[string]any{
				"cliPath": name,
				"args":    args,
			}
		},
	})
	defer resetExecCommander()

	_, err := DiffRefs("from", "to", DiffOptions{CliPath: "git", CliWd: "."})
	assert.NoError(t, err)
	assert.Equal(t, execCommanderCalledWith["cliPath"], "git")
	for _, v := range []string{"diff", "from", "to"} {
		assert.Contains(t, execCommanderCalledWith["args"], v)
	}

	mockCmd.AssertExpectations(t)
}

func TestDiffStaged(t *testing.T) {
	var execCommanderCalledWith map[string]any
	mockCmd, resetExecCommander := newMockedExecCommander(mockedExecCommanderOptions{
		setDirCalledWith: ".",
		onExecCommander: func(name string, args ...string) {
			execCommanderCalledWith = map[string]any{
				"cliPath": name,
				"args":    args,
			}
		},
	})
	defer resetExecCommander()

	_, err := DiffStaged(DiffOptions{CliPath: "git", CliWd: "."})
	assert.NoError(t, err)
	assert.Equal(t, execCommanderCalledWith["cliPath"], "git")
	for _, v := range []string{"diff", "--cached"} {
		assert.Contains(t, execCommanderCalledWith["args"], v)
	}

	mockCmd.AssertExpectations(t)
}

func TestDiffCommit(t *testing.T) {
	var execCommanderCalledWith map[string]any
	mockCmd, resetExecCommander := newMockedExecCommander(mockedExecCommanderOptions{
		setDirCalledWith: ".",
		onExecCommander: func(name string, args ...string) {
			execCommanderCalledWith = map[string]any{
				"cliPath": name,
				"args":    args,
			}
		},
	})
	defer resetExecCommander()

	_, err := DiffCommit("ref", DiffOptions{CliPath: "git", CliWd: "."})
	assert.NoError(t, err)
	assert.Equal(t, execCommanderCalledWith["cliPath"], "git")
	for _, v := range []string{"show", "ref"} {
		assert.Contains(t, execCommanderCalledWith["args"], v)
	}

	mockCmd.AssertExpectations(t)
}

func TestRunCli_Success(t *testing.T) {
	mockCmd, resetExecCommander := newMockedExecCommander(mockedExecCommanderOptions{
		setDirCalledWith:  "testdir",
		combinedOutputOut: []byte("mock output"),
		getArgsOut:        []string{"arg1", "arg2"},
	})
	defer resetExecCommander()

	res, err := runCli("git", "testdir", "arg1", "arg2")
	assert.NoError(t, err)
	assert.Equal(t, []byte("mock output"), res.Out)
	assert.Equal(t, "arg1 arg2", res.FullCommand)

	mockCmd.AssertExpectations(t)
}

func TestRunCli_Error(t *testing.T) {
	mockCmd, resetExecCommander := newMockedExecCommander(mockedExecCommanderOptions{
		setDirCalledWith:  "dir",
		combinedOutputErr: errors.New("boom"),
		getArgsOut:        []string{},
	})
	defer resetExecCommander()

	_, err := runCli("cli", "dir", "fail")
	assert.Error(t, err)
	mockCmd.AssertExpectations(t)
}
