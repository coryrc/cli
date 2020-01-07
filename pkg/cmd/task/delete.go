// Copyright © 2019 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package task

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/names"
	"github.com/tektoncd/cli/pkg/helper/options"
	validate "github.com/tektoncd/cli/pkg/helper/validate"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "task", ForceDelete: false, DeleteAll: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete Tasks with names 'foo' and 'bar' in namespace 'quux':

    tkn task delete foo bar -n quux

or

    tkn t rm foo bar -n quux
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete task resources in a namespace",
		Example:      eg,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			if err := opts.CheckOptions(s, args); err != nil {
				return err
			}

			return deleteTask(opts, s, p, args)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAll, "all", "a", false, "Whether to delete related resources (taskruns) (default: false)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_task")
	return c
}

func deleteTask(opts *options.DeleteOptions, s *cli.Stream, p cli.Params, tNames []string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	var errs []error
	addPrintErr := func(err error) {
		errs = append(errs, err)
		fmt.Fprintf(s.Err, "%s\n", err)
	}

	var successfulTasks []string
	var successfulTaskRuns []string

	for _, tName := range tNames {
		if err := cs.Tekton.TektonV1alpha1().Tasks(p.Namespace()).Delete(tName, &metav1.DeleteOptions{}); err != nil {
			addPrintErr(fmt.Errorf("failed to delete task %q: %s", tName, err))
			continue
		}
		successfulTasks = append(successfulTasks, tName)

		if !opts.DeleteAll {
			continue
		}

		lOpts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/task=%s", tName),
		}

		taskRuns, err := cs.Tekton.TektonV1alpha1().TaskRuns(p.Namespace()).List(lOpts)
		if err != nil {
			addPrintErr(err)
			continue
		}

		for _, tr := range taskRuns.Items {
			if err := cs.Tekton.TektonV1alpha1().TaskRuns(p.Namespace()).Delete(tr.Name, &metav1.DeleteOptions{}); err != nil {
				addPrintErr(fmt.Errorf("failed to delete taskrun %q: %s", tr.Name, err))
				continue
			}
			successfulTaskRuns = append(successfulTaskRuns, tr.Name)
		}
	}

	if len(successfulTaskRuns) > 0 {
		fmt.Fprintf(s.Out, "TaskRuns deleted: %s\n", names.QuotedList(successfulTaskRuns))
	}
	if len(successfulTasks) > 0 {
		fmt.Fprintf(s.Out, "Tasks deleted: %s\n", names.QuotedList(successfulTasks))
	}

	return multierr.Combine(errs...)
}
