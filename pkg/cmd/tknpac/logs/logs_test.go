package logs

import (
	"testing"

	"github.com/jonboulle/clockwork"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/apis/pipelinesascode/v1alpha1"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/cli"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/consoleui"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/params"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/params/clients"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/params/info"
	tcli "github.com/openshift-pipelines/pipelines-as-code/pkg/test/cli"
	testclient "github.com/openshift-pipelines/pipelines-as-code/pkg/test/clients"
	tektontest "github.com/openshift-pipelines/pipelines-as-code/pkg/test/tekton"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestLogs(t *testing.T) {
	cw := clockwork.NewFakeClock()
	ns := "ns"
	completed := tektonv1beta1.PipelineRunReasonCompleted.String()

	tests := []struct {
		name             string
		wantErr          bool
		repoName         string
		currentNamespace string
		statuses         []v1alpha1.RepositoryRunStatus
		shift            int
		pruns            []*tektonv1beta1.PipelineRun
	}{
		{
			name:             "good/show logs",
			wantErr:          false,
			repoName:         "test",
			currentNamespace: ns,
			shift:            1,
			statuses: []v1alpha1.RepositoryRunStatus{
				{
					PipelineRunName: "test-pipeline",
				},
			},
			pruns: []*tektonv1beta1.PipelineRun{
				tektontest.MakePRCompletion(cw, "test-pipeline", ns, completed, map[string]string{}, 30),
			},
		},
		{
			name:             "bad/shift",
			wantErr:          true,
			repoName:         "test",
			currentNamespace: ns,
			shift:            2,
			statuses: []v1alpha1.RepositoryRunStatus{
				{
					PipelineRunName: "test-pipeline",
				},
			},
			pruns: []*tektonv1beta1.PipelineRun{
				tektontest.MakePRCompletion(cw, "test-pipeline", ns, completed, map[string]string{}, 30),
			},
		},
		{
			name:             "bad/no status",
			wantErr:          true,
			repoName:         "test",
			currentNamespace: ns,
			shift:            2,
			pruns: []*tektonv1beta1.PipelineRun{
				tektontest.MakePRCompletion(cw, "test-pipeline", ns, completed, map[string]string{}, 30),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositories := []*v1alpha1.Repository{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tt.repoName,
						Namespace: tt.currentNamespace,
					},
					Spec: v1alpha1.RepositorySpec{
						URL: "https://anurl.com",
					},
					Status: tt.statuses,
				},
			}
			tdata := testclient.Data{
				Namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: tt.currentNamespace,
						},
					},
				},
				PipelineRuns: tt.pruns,
				Repositories: repositories,
			}

			ctx, _ := rtesting.SetupFakeContext(t)
			stdata, _ := testclient.SeedTestData(t, ctx, tdata)
			cs := &params.Run{
				Clients: clients.Clients{
					PipelineAsCode: stdata.PipelineAsCode,
					Tekton:         stdata.Pipeline,
					ConsoleUI:      consoleui.FallBackConsole{},
				},
				Info: info.Info{Kube: info.KubeOpts{Namespace: tt.currentNamespace}},
			}
			io, _ := tcli.NewIOStream()
			lopts := &logOption{
				cs: cs,
				opts: &cli.PacCliOpts{
					Namespace: tt.currentNamespace,
				},
				repoName:  tt.repoName,
				shift:     tt.shift,
				tknPath:   "/bin/true",
				ioStreams: io,
			}

			err := log(ctx, lopts)
			if tt.wantErr {
				if err == nil {
					t.Errorf("log() wantError is true but no error has been set")
				}
			}
		})
	}
}
