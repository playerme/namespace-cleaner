package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"time"

	"k8s.io/api/core/v1"

	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/helm/pkg/helm"
)

const (
	tillerHost = "tiller-deploy.kube-system:44134"
)

func authIncluster() *rest.Config {
	config, err := rest.InClusterConfig()

	if err != nil {
		panic(err.Error())
	}

	return config
}

func authLocal() *rest.Config {
	// //testing locally

	var kubeconfig *string
	if home := homeDir(); home != "" {
		fmt.Println(filepath.Join(home, ".kube", "config"))
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Println("error build config")
		panic(err.Error())
	}

	// create the clientset
	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {

	// 	fmt.Println("error config")
	// 	panic(err.Error())
	// }

	return config
}

func cleanupHelms(revnamespaces []string) {
	for _, revnamespace := range revnamespaces {
		cleanupHelm(revnamespace)
	}
}

//clean up using helm
func cleanupHelm(revnamespace string) {

	helmClient := helm.NewClient(helm.Host(tillerHost))
	Releases, err := helmClient.ListReleases(helm.ReleaseListFilter(revnamespace))

	release := Releases.GetReleases()
	for _, r := range release {

		fmt.Println("purge release: " + r.GetName())
		helmClient.DeleteRelease(r.GetName(), helm.DeletePurge(true))
	}
	if err != nil {
		log.Fatalf("err: %v", err)
	}
}

func listNamespace(clientset *kubernetes.Clientset) *v1.NamespaceList {
	namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		fmt.Println("namespace")
		panic(err.Error())
	}
	return namespaces
}

func deleteNamespace(clientset *kubernetes.Clientset, revnamespaces []string) {
	deletePolicy := metav1.DeletePropagationBackground
	orphandependent := false
	for _, deletenamespace := range revnamespaces {
		fmt.Println("deleting namespace: " + deletenamespace)
		err := clientset.CoreV1().Namespaces().Delete(deletenamespace, &metav1.DeleteOptions{

			OrphanDependents:  &orphandependent,
			PropagationPolicy: &deletePolicy,
		})

		if err != nil {
			fmt.Println(err)
		}
	}

}

func getenv(key string, fallback bool) bool {
	value := os.Getenv(key)

	if len(value) == 0 {

		return fallback
	}
	if value == "true" {
		return true
	} else if value == "false" {
		return false
	}
	return fallback
}

func main() {

	exempts := os.Getenv("EXEMPTION")
	helm := getenv("HELM", false)
	
	fmt.Printf("helm is true '%t'", helm)
	
	duration, err := time.ParseDuration(os.Getenv("DURATION"))

	if err != nil {
		fmt.Println("Error Duration is empty")
		panic(err)
	}

	config := authIncluster()
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	//get environment variable duration

	//list environment excemption comma separated
	exemptList := strings.Split(exempts, ",")

	if exempts == "" {
		fmt.Println("Error Exemption is empty")
		panic(err)
	}

	//tags := os.Getenv("TAG")

	tag := "rev"

	fmt.Println("Duration: " + duration.String())
	fmt.Println("Exemptions: ")
	fmt.Println(exemptList)

	exempt := "rev-master"
	for {

		var revnamespaces []string
		now := time.Now()
		//set time in minutes

		namespaces := listNamespace(clientset)

		for _, namespace := range namespaces.Items {

			if strings.Contains(namespace.Name, tag) && namespace.Name != exempt {
				fmt.Println(namespace.Name + " : " + now.Sub(namespace.CreationTimestamp.Time).String())

				if duration.Nanoseconds() <= now.Sub(namespace.CreationTimestamp.Time).Nanoseconds() {
					revnamespaces = append(revnamespaces, namespace.Name)
					fmt.Println(namespace.Name + " is pass " + duration.String())
					fmt.Println(now.Sub(namespace.CreationTimestamp.Time))
				}
			}

		}

		revnamespaces = difference(revnamespaces, exemptList)

		if helm == true {

			cleanupHelms(revnamespaces)
		}
		deleteNamespace(clientset, revnamespaces)

		fmt.Println(".....END")
		time.Sleep(60 * time.Second)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {

		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func difference(toRemoval []string, toSave []string) []string {
	var remove []string

	for _, checkRemoval := range toRemoval {
		found := false
		for _, save := range toSave {
			save = strings.TrimSpace(save)
			if checkRemoval == save {
				found = true

				break
			}

		}
		if !found {
			remove = append(remove, checkRemoval)
			fmt.Println(remove)
		}
	}

	return remove

}
