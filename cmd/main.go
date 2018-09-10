package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	kubecmd "k8s.io/kubernetes/pkg/kubectl/cmd"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"
	"k8s.io/kubernetes/pkg/kubectl/scheme"

	"github.com/openshift/api"
	"github.com/openshift/builder/pkg/cmd/infra/builder"
	"github.com/openshift/library-go/pkg/serviceability"
	"github.com/openshift/origin/pkg/api/install"
	"github.com/openshift/origin/pkg/api/legacy"
	"github.com/openshift/origin/pkg/cmd/flagtypes"
	"github.com/openshift/origin/pkg/cmd/recycle"
	"github.com/openshift/origin/pkg/cmd/templates"
	"github.com/openshift/origin/pkg/cmd/util/term"
	"github.com/openshift/origin/pkg/oc/util/ocscheme"
	"github.com/openshift/origin/pkg/version"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"), version.Get())()
	defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()

	rand.Seed(time.Now().UTC().UnixNano())
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	// the kubectl scheme expects to have all the recognizable external types it needs to consume.  Install those here.
	api.Install(scheme.Scheme)
	legacy.InstallExternalLegacyAll(scheme.Scheme)

	// the legacyscheme is used in kubectl and expects to have the internal types registered.  Explicitly wire our types here.
	// this does
	install.InstallInternalOpenShift(legacyscheme.Scheme)
	legacy.InstallInternalLegacyAll(scheme.Scheme)

	basename := filepath.Base(os.Args[0])
	command := cli.CommandFor(basename)
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}

// CommandFor returns the appropriate command for this base name,
// or the OpenShift CLI command.
func CommandFor(basename string) *cobra.Command {
	var cmd *cobra.Command

	in, out, errout := os.Stdin, os.Stdout, os.Stderr

	switch basename {
	case "openshift-sti-build":
		cmd = builder.NewCommandS2IBuilder(basename)
	case "openshift-docker-build":
		cmd = builder.NewCommandDockerBuilder(basename)
	case "openshift-git-clone":
		cmd = builder.NewCommandGitClone(basename)
	case "openshift-manage-dockerfile":
		cmd = builder.NewCommandManageDockerfile(basename)
	case "openshift-extract-image-content":
		cmd = builder.NewCommandExtractImageContent(basename)
	default:
		fmt.Println("unknown command name: %s", basename)
		os.Exit(1)
	}

	flagtypes.GLog(cmd.PersistentFlags())

	return cmd
}
