package main

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	//"flag"
	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/tools/clientcmd"
	//"path/filepath"
	"os"
	"strings"
)

func main() {

	config, err := rest.InClusterConfig()

	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	// //testing locally
	// var kubeconfig *string
	// if home := homeDir(); home != "" {
	// 	fmt.Println(filepath.Join(home, ".kube", "config"))
	// 	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	// } else {
	// 	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	// }
	// flag.Parse()

	// // use the current context in kubeconfig
	// config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	// if err != nil {
	// 	fmt.Println("error build config")
	// 	panic(err.Error())
	// }

	// // create the clientset
	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {

	// 	fmt.Println("error config")
	// 	panic(err.Error())
	// }

	//get environment variable duration
	duration, err := time.ParseDuration(os.Getenv("DURATION"))

	//list environment excemption comma separated

	if err != nil {
		fmt.Println("Error Duration is empty")
		panic(err)
	}

	exempts := os.Getenv("EXEMPTION")

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
		namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			fmt.Println("namespace")
			panic(err.Error())
		}
		var revnamespaces []string
		now := time.Now()
		//set time in minutes

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
		deletePolicy := metav1.DeletePropagationBackground
		orphandependent := false
		for _, deletenamespace := range revnamespaces {
			fmt.Println("deleting namespace: " + deletenamespace)
			err = clientset.CoreV1().Namespaces().Delete(deletenamespace, &metav1.DeleteOptions{

				OrphanDependents:  &orphandependent,
				PropagationPolicy: &deletePolicy,
			})
		}

		if err != nil {
			fmt.Println(err)
		}
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