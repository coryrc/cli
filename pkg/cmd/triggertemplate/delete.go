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

package triggertemplate

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/names"
	"github.com/tektoncd/cli/pkg/helper/options"
	"github.com/tektoncd/cli/pkg/helper/validate"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "triggertemplate", ForceDelete: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete TriggerTemplates with names 'foo' and 'bar' in namespace 'quux'

    tkn triggertemplate delete foo bar -n quux

or

    tkn tt rm foo bar -n quux
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete triggertemplates in a namespace",
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

			return deleteTriggerTemplates(s, p, args)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_triggertemplate")
	return c
}

func deleteTriggerTemplates(s *cli.Stream, p cli.Params, ttNames []string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	var errs []error
	var success []string

	for _, ttName := range ttNames {
		if err := cs.Triggers.TektonV1alpha1().TriggerTemplates(p.Namespace()).Delete(ttName, &metav1.DeleteOptions{}); err != nil {
			err = fmt.Errorf("failed to delete triggertemplate %q: %s", ttName, err)
			errs = append(errs, err)
			fmt.Fprintf(s.Err, "%s\n", err)
			continue
		}
		success = append(success, ttName)
	}
	if len(success) > 0 {
		fmt.Fprintf(s.Out, "TriggerTemplates deleted: %s\n", names.QuotedList(success))
	}

	return multierr.Combine(errs...)
}
