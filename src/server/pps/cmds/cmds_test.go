// TODO(msteffen) Add tests for:
//
// - restart datum
// - stop job
// - delete job
//
// - inspect job
// - list job
//
// - create pipeline
// - create pipeline --push-images (re-enable existing test)
// - update pipeline
// - delete pipeline
//
// - inspect pipeline
// - list pipeline
//
// - start pipeline
// - stop pipeline
//
// - list datum
// - inspect datum
// - logs

package cmds

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/pachyderm/pachyderm/v2/src/client"
	"github.com/pachyderm/pachyderm/v2/src/internal/errors"
	"github.com/pachyderm/pachyderm/v2/src/internal/minikubetestenv"
	"github.com/pachyderm/pachyderm/v2/src/internal/require"
	tu "github.com/pachyderm/pachyderm/v2/src/internal/testutil"
)

const badJSON1 = `
{
"356weryt

}
`

const badJSON2 = `{
{
    "a": 1,
    "b": [23,4,4,64,56,36,7456,7],
    "c": {"e,f,g,h,j,j},
    "d": 3452.36456,
}
`

func TestSyntaxErrorsReportedCreatePipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		echo -n '{{.badJSON1}}' \
		  | ( pachctl create pipeline -f - 2>&1 || true ) \
		  | match "malformed pipeline spec"

		echo -n '{{.badJSON2}}' \
		  | ( pachctl create pipeline -f - 2>&1 || true ) \
		  | match "malformed pipeline spec"
		`,
		"badJSON1", badJSON1,
		"badJSON2", badJSON2,
	).Run())
}

func TestSyntaxErrorsReportedUpdatePipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		echo -n '{{.badJSON1}}' \
		  | ( pachctl update pipeline -f - 2>&1 || true ) \
		  | match "malformed pipeline spec"

		echo -n '{{.badJSON2}}' \
		  | ( pachctl update pipeline -f - 2>&1 || true ) \
		  | match "malformed pipeline spec"
		`,
		"badJSON1", badJSON1,
		"badJSON2", badJSON2,
	).Run())
}

func TestRawFullPipelineInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
	`).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		pachctl create repo data
		pachctl put file data@master:/file <<<"This is a test"
		pachctl create pipeline <<EOF
		  {
		    "pipeline": {"name": "{{.pipeline}}"},
		    "input": {
		      "pfs": {
		        "glob": "/*",
		        "repo": "data"
		      }
		    },
		    "transform": {
		      "cmd": ["bash"],
		      "stdin": ["cp /pfs/data/file /pfs/out"]
		    }
		  }
		EOF
		`,
		"pipeline", tu.UniqueString("p-")).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		pachctl wait commit data@master

		# make sure the results have the full pipeline info, including version
		pachctl list job --raw \
			| match "pipeline_version"
		`).Run())
}

// TestJSONMultiplePipelines tests that pipeline specs with multiple pipelines
// are accepted, as long as they're separated by a YAML document separator
func TestJSONMultiplePipelines(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
		pachctl create repo input
		pachctl create pipeline -f - <<EOF
		{
		  "pipeline": {
		    "name": "first"
		  },
		  "input": {
		    "pfs": {
		      "glob": "/*",
		      "repo": "input"
		    }
		  },
		  "transform": {
		    "cmd": [ "/bin/bash" ],
		    "stdin": [
		      "cp /pfs/input/* /pfs/out"
		    ]
		  }
		}
		---
		{
		  "pipeline": {
		    "name": "second"
		  },
		  "input": {
		    "pfs": {
		      "glob": "/*",
		      "repo": "first"
		    }
		  },
		  "transform": {
		    "cmd": [ "/bin/bash" ],
		    "stdin": [
		      "cp /pfs/first/* /pfs/out"
		    ]
		  }
		}
		EOF

		pachctl start commit input@master
		echo foo | pachctl put file input@master:/foo
		echo bar | pachctl put file input@master:/bar
		echo baz | pachctl put file input@master:/baz
		pachctl finish commit input@master
		pachctl wait commit second@master
		pachctl get file second@master:/foo | match foo
		pachctl get file second@master:/bar | match bar
		pachctl get file second@master:/baz | match baz
		`,
	).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `pachctl list pipeline`).Run())
}

// TestJSONStringifiedNumberstests that JSON pipelines may use strings to
// specify numeric values such as a pipeline's parallelism (a feature of gogo's
// JSON parser).
func TestJSONStringifiedNumbers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
		pachctl create repo input
		pachctl create pipeline -f - <<EOF
		{
		  "pipeline": {
		    "name": "first"
		  },
		  "input": {
		    "pfs": {
		      "glob": "/*",
		      "repo": "input"
		    }
		  },
		  "parallelism_spec": {
		    "constant": "1"
		  },
		  "transform": {
		    "cmd": [ "/bin/bash" ],
		    "stdin": [
		      "cp /pfs/input/* /pfs/out"
		    ]
		  }
		}
		EOF

		pachctl start commit input@master
		echo foo | pachctl put file input@master:/foo
		echo bar | pachctl put file input@master:/bar
		echo baz | pachctl put file input@master:/baz
		pachctl finish commit input@master
		pachctl wait commit first@master
		pachctl get file first@master:/foo | match foo
		pachctl get file first@master:/bar | match bar
		pachctl get file first@master:/baz | match baz
		`,
	).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `pachctl list pipeline`).Run())
}

func TestRunPipeline(t *testing.T) {
	// TODO(2.0 optional): Run pipeline creates a dangling commit, but uses the stats branch for stats commits.
	// Since there is no relationship between the commits created by run pipeline, there should be no
	// relationship between the stats commits. So, the stats commits should also be dangling commits.
	// This might be easier to address when global IDs are implemented.
	t.Skip("Run pipeline does not work correctly with stats enabled")
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	pipeline := tu.UniqueString("p-")
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
	`).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		pachctl create repo data

		# Create a commit and put some data in it
		commit1=$(pachctl start commit data@master)
		echo "file contents" | pachctl put file data@${commit1}:/file -f -
		pachctl finish commit data@${commit1}

		pachctl put file data@master:/file <<<"This is a test"
		pachctl create pipeline <<EOF
			{
			  "pipeline": {"name": "{{.pipeline}}"},
			  "input": {
			    "pfs": {
			      "glob": "/*",
			      "repo": "data"
			    }
			  },
			  "transform": {
			    "cmd": ["bash"],
			    "stdin": ["cp /pfs/data/file /pfs/out"]
			  }
			}
		EOF

		# make sure the run pipeline command accepts various provenance formats
		pachctl run pipeline {{.pipeline}} data@${commit1}
		pachctl run pipeline {{.pipeline}} data@master=${commit1}
		pachctl run pipeline {{.pipeline}} data@master
		`,
		"pipeline", pipeline).Run())
}

// TestYAMLPipelineSpec tests creating a pipeline with a YAML pipeline spec
func TestYAMLPipelineSpec(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	// Note that BashCmd dedents all lines below including the YAML (which
	// wouldn't parse otherwise)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
		pachctl create repo input
		pachctl create pipeline -f - <<EOF
		pipeline:
		  name: first
		input:
		  pfs:
		    glob: /*
		    repo: input
		transform:
		  cmd: [ /bin/bash ]
		  stdin:
		    - "cp /pfs/input/* /pfs/out"
		---
		pipeline:
		  name: second
		input:
		  pfs:
		    glob: /*
		    repo: first
		transform:
		  cmd: [ /bin/bash ]
		  stdin:
		    - "cp /pfs/first/* /pfs/out"
		EOF
		pachctl start commit input@master
		echo foo | pachctl put file input@master:/foo
		echo bar | pachctl put file input@master:/bar
		echo baz | pachctl put file input@master:/baz
		pachctl finish commit input@master
		pachctl wait commit second@master
		pachctl get file second@master:/foo | match foo
		pachctl get file second@master:/bar | match bar
		pachctl get file second@master:/baz | match baz
		`,
	).Run())
}

func TestListPipelineFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
	`).Run())
	pipeline := tu.UniqueString("pipeline")
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
		pachctl create repo input
		pachctl create pipeline -f - <<EOF
		{
		"pipeline": {
		  "name": "{{.pipeline}}"
		},
		"input": {
		  "pfs": {
		    "glob": "/*",
		    "repo": "input"
		  }
		},
		  "parallelism_spec": {
		    "constant": "1"
		  },
		"transform": {
		  "cmd": [ "/bin/bash" ],
		  "stdin": [
		    "cp /pfs/input/* /pfs/out"
		  ]
		}
		}
		EOF

		echo foo | pachctl put file input@master:/foo
		pachctl wait commit {{.pipeline}}@master
		# make sure we see the pipeline with the appropriate state filters
		pachctl list pipeline | match {{.pipeline}}
		pachctl list pipeline --state crashing --state failure | match -v {{.pipeline}}
		pachctl list pipeline --state starting --state running | match {{.pipeline}}
	`, "pipeline", pipeline,
	).Run())
}

// TestYAMLError tests that when creating pipelines using a YAML spec with an
// error, you get an error indicating the problem in the YAML, rather than an
// error complaining about multiple documents.
//
// Note that because we parse, serialize, and re-parse pipeline specs to work
// around protobuf-generated struct tags and such, the error will refer to
// "json" (the format used for the canonicalized pipeline), but the issue
// referenced by the error (use of a string instead of an array for 'cmd') is
// the main problem below
func TestYAMLError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	// "cmd" should be a list, instead of a string
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
		pachctl create repo input
		( pachctl create pipeline -f - 2>&1 <<EOF || true
		pipeline:
		  name: first
		input:
		  pfs:
		    glob: /*
		    repo: input
		transform:
		  cmd: /bin/bash # should be list, instead of string
		  stdin:
		    - "cp /pfs/input/* /pfs/out"
		EOF
		) | match "cannot unmarshal string into Go value of type \[\]json.RawMessage"
		`,
	).Run())
}

// TestYAMLSecret tests creating a YAML pipeline with a secret (i.e. the fix for
// https://github.com/pachyderm/pachyderm/v2/issues/4119)
func TestYAMLSecret(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, ns := minikubetestenv.AcquireCluster(t)
	// Note that BashCmd dedents all lines below including the YAML (which
	// wouldn't parse otherwise)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all

		# kubectl get secrets >&2
		kubectl delete secrets/test-yaml-secret -n {{ .namespace }} || true
		kubectl create secret generic test-yaml-secret --from-literal=my-key=my-value -n {{ .namespace }}

		pachctl create repo input
		pachctl put file input@master:/foo <<<"foo"
		pachctl create pipeline -f - <<EOF
		  pipeline:
		    name: pipeline
		  input:
		    pfs:
		      glob: /*
		      repo: input
		  transform:
		    cmd: [ /bin/bash ]
		    stdin:
		      - "env | grep MY_SECRET >/pfs/out/vars"
		    secrets:
		      - name: test-yaml-secret
		        env_var: MY_SECRET
		        key: my-key
		EOF
		pachctl wait commit pipeline@master
		pachctl get file pipeline@master:/vars | match MY_SECRET=my-value
		`, "namespace", ns,
	).Run())
}

// TestYAMLTimestamp tests creating a YAML pipeline with a timestamp (i.e. the
// fix for https://github.com/pachyderm/pachyderm/v2/issues/4209)
func TestYAMLTimestamp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	// Note that BashCmd dedents all lines below including the YAML (which
	// wouldn't parse otherwise)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all

		# If the pipeline comes up without error, then the YAML parsed
		pachctl create pipeline -f - <<EOF
		  pipeline:
		    name: pipeline
		  input:
		    cron:
		      name: in
		      start: "2019-10-10T22:30:05Z"
		      spec: "@yearly"
		  transform:
		    cmd: [ /bin/bash ]
		    stdin:
		      - "cp /pfs/in/* /pfs/out"
		EOF
		pachctl list pipeline | match 'pipeline'
		`,
	).Run())
}

func TestEditPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
	`).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		pachctl create repo data
		pachctl create pipeline <<EOF
		  pipeline:
		    name: my-pipeline
		  input:
		    pfs:
		      glob: /*
		      repo: data
		  transform:
		    cmd: [ /bin/bash ]
		    stdin:
		      - "cp /pfs/data/* /pfs/out"
		EOF
		`).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		EDITOR="cat -u" pachctl edit pipeline my-pipeline -o yaml \
		| match 'name: my-pipeline' \
		| match 'repo: data' \
		| match 'cmd:' \
		| match 'cp /pfs/data/\* /pfs/out'
		`).Run())
}

func TestMissingPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	// should fail because there's no pipeline object in the spec
	require.YesError(t, tu.PachctlBashCmd(t, c, `
		pachctl create pipeline <<EOF
		  {
		    "transform": {
			  "image": "ubuntu:20.04"
		    },
		    "input": {
		      "pfs": {
		        "repo": "in",
		        "glob": "/*"
		      }
		    }
		  }
		EOF
	`).Run())
}

func TestUnnamedPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	// should fail because there's no pipeline name
	require.YesError(t, tu.PachctlBashCmd(t, c, `
		pachctl create pipeline <<EOF
		  {
		    "pipeline": {},
		    "transform": {
			  "image": "ubuntu:20.04"
		    },
		    "input": {
		      "pfs": {
		        "repo": "in",
		        "glob": "/*"
		      }
		    }
		  }
		EOF
	`).Run())
}

// func TestPushImages(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration tests in short mode")
// 	}
// 	ioutil.WriteFile("test-push-images.json", []byte(`{
//   "pipeline": {
//     "name": "test_push_images"
//   },
//   "transform": {
//     "cmd": [ "true" ],
// 	"image": "test-job-shim"
//   }
// }`), 0644)
// 	os.Args = []string{"pachctl", "create", "pipeline", "--push-images", "-f", "test-push-images.json"}
// 	require.NoError(t, rootCmd().Execute())
// }

func runPipelineWithImageGetStderr(t *testing.T, c *client.APIClient, image string) (string, error) {
	// reset and create some test input
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
		pachctl create repo in
		echo "foo" | pachctl put file in@master:/file1
		echo "bar" | pachctl put file in@master:/file2
	`).Run())

	cmd := tu.PachctlBashCmd(t, c, `
		pachctl create pipeline <<EOF
		  {
		    "pipeline": {
		      "name": "first"
		    },
		    "transform": {
		      "image": "{{.image}}"
		    },
		    "input": {
		      "pfs": {
		        "repo": "in",
		        "glob": "/*"
		      }
		    }
		  }
		EOF
	`, "image", image)
	buf := &bytes.Buffer{}
	cmd.Stderr = buf
	// cmd.Stdout = os.Stdout // uncomment for debugging
	cmd.Env = os.Environ()
	err := cmd.Run()
	stderr := buf.String()
	return stderr, errors.EnsureStack(err)
}

func TestWarningLatestTag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	// should emit a warning because user specified latest tag on docker image
	stderr, err := runPipelineWithImageGetStderr(t, c, "ubuntu:latest")
	require.NoError(t, err, "%v", err)
	require.Matches(t, "WARNING", stderr)
}

func TestWarningEmptyTag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	// should emit a warning because user specified empty tag, equivalent to
	// :latest
	stderr, err := runPipelineWithImageGetStderr(t, c, "ubuntu")
	require.NoError(t, err)
	require.Matches(t, "WARNING", stderr)
}

func TestNoWarningTagSpecified(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	// should not emit a warning (stderr should be empty) because user
	// specified non-empty, non-latest tag
	stderr, err := runPipelineWithImageGetStderr(t, c, "ubuntu:20.04")
	require.NoError(t, err)
	require.Equal(t, "", stderr)
}

func TestPipelineCrashingRecovers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
	`).Run())

	// prevent new pods from being scheduled
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		kubectl cordon minikube
	`).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		pachctl create repo data
		pachctl create pipeline <<EOF
		  pipeline:
		    name: my-pipeline
		  input:
		    pfs:
		      glob: /*
		      repo: data
		  transform:
		    cmd: [ /bin/bash ]
		    stdin:
		      - "cp /pfs/data/* /pfs/out"
		EOF
		`).Run())

	require.NoErrorWithinTRetry(t, 30*time.Second, func() error {
		return errors.EnsureStack(tu.PachctlBashCmd(t, c, `
		pachctl list pipeline \
		| match my-pipeline \
		| match crashing
		`).Run())
	})

	// allow new pods to be scheduled
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		kubectl uncordon minikube
	`).Run())

	require.NoErrorWithinTRetry(t, 30*time.Second, func() error {
		return errors.EnsureStack(tu.PachctlBashCmd(t, c, `
		pachctl list pipeline \
		| match my-pipeline \
		| match running
		`).Run())
	})
}

func TestJsonnetPipelineTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
	`).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		pachctl create repo data
		pachctl put file data@master:/foo <<<"foo-data"
		pachctl create pipeline --jsonnet - --arg name=foo --arg output=bar <<EOF
		function(name, output) {
		  pipeline: { name: name+"-pipeline" },
		  input: {
		    pfs: {
		      name: "input",
		      glob: "/*",
		      repo: "data"
		    }
		  },
		  transform: {
		    cmd: [ "/bin/bash" ],
		    stdin: [ "cp /pfs/input/* /pfs/out/"+output ]
		  }
		}
		EOF
		pachctl list pipeline | match foo-pipeline
		pachctl wait commit foo-pipeline@master
		pachctl get file foo-pipeline@master:/bar | match foo-data
		`).Run())
}

func TestJsonnetPipelineTemplateMulti(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
	`).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		pachctl create repo data
		pachctl put file data@master:/foo <<<"foo-data"
		pachctl create pipeline --jsonnet - \
		  --arg firstname=foo --arg lastname=bar --arg output=baz <<EOF
		function(firstname, lastname, output) [ {
		  pipeline: { name: firstname+"-pipeline" },
		  input: {
		    pfs: {
		      name: "input",
		      glob: "/*",
		      repo: "data"
		    }
		  },
		  transform: {
		    cmd: [ "/bin/bash" ],
		    stdin: [ "cp /pfs/input/* /pfs/out/"+output+"-middle" ]
		  }
		}, {
		  pipeline: { name: lastname+"-pipeline" },
		  input: {
		    pfs: {
		      name: "input",
		      glob: "/*",
		      repo: firstname+"-pipeline"
		    }
		  },
		  transform: {
		    cmd: [ "/bin/bash" ],
		    stdin: [ "cp /pfs/input/"+output+"-middle /pfs/out/"+output+"-final" ]
		  }
		} ]
		EOF
		pachctl list pipeline | match foo-pipeline | match bar-pipeline
		pachctl wait commit bar-pipeline@master
		pachctl get file foo-pipeline@master:/baz-middle | match foo-data
		pachctl get file bar-pipeline@master:/baz-final | match foo-data
		`).Run())
}

func TestJsonnetPipelineTemplateError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		yes | pachctl delete all
	`).Run())
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		pachctl create repo data
		pachctl put file data@master:/foo <<<"foo-data"
		# expect 'create pipeline' to fail, but still run 'match'
		set +e +o pipefail
		# create a pipeline, and confirm that error message appears on stderr
		# Note: discard stdout, and send stderr on stdout instead
		createpipeline_stderr="$(
		  pachctl create pipeline \
		    --jsonnet - --arg parallelism=NaN 2>&1 >/dev/null <<EOF
		function(parallelism) {
		  pipeline: { name: "pipeline" },
		  input: {
		    pfs: {
		      name: "input",
		      glob: "/*",
		      repo: "data"
		    }
		  },
		  parallelism_spec: {
		    constant: std.parseInt(parallelism)
		  },
		  transform: {
		    cmd: [ "/bin/bash" ],
		    stdin: [ "cp /pfs/input/* /pfs/out/" ]
		  }
		}
		EOF
		  )"
		createpipeline_status="$?"
		# reset error options before testing output
		set -e -o pipefail
		echo "${createpipeline_status}" | match -v "0"
		echo "${createpipeline_stderr}" | match 'NaN is not a base 10 integer'
		pachctl list pipeline | match -v foo-pipeline
		`).Run())
}

func TestJobDatumCount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	c, _ := minikubetestenv.AcquireCluster(t)
	require.NoError(t, tu.PachctlBashCmd(t, c, `
		pachctl create repo data
		pachctl put file data@master:/foo <<<"foo-data"
		pachctl create pipeline <<EOF
		{
		    "pipeline": {"name": "{{.pipeline}}"},
		    "input": {
		      "pfs": {
		        "glob": "/*",
		        "repo": "data"
		      }
		    },
		    "transform": {
		      "cmd": ["bash"],
		      "stdin": ["cp -rp /pfs/data/ /pfs/out"]
		    }
		  }
		EOF
	`, "pipeline", tu.UniqueString("p-")).Run())
	// need some time for the pod to spin up
	require.NoErrorWithinTRetry(t, 2*time.Minute, func() error {
		//nolint:wrapcheck
		return tu.PachctlBashCmd(t, c, `
pachctl list job -x | match ' / 1'
`).Run()
	}, "expected to see one datum")
	require.NoError(t, tu.PachctlBashCmd(t, c, `pachctl put file data@master:/bar <<<"bar-data"`).Run())
	// with the new datum, should see the pipeline run another job with two datums

	require.NoErrorWithinTRetry(t, 2*time.Minute, func() error {
		//nolint:wrapcheck
		return tu.PachctlBashCmd(t, c, `
pachctl list job -x | match ' / 2'
`).Run()
	}, "expected to see two datums")
}
