package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func Namespace() string {
	// This way assumes you've set the POD_NAMESPACE environment variable using the downward API.
	// This check has to be done first for backwards compatibility with the way InClusterConfig was originally set up
	if ns, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return ns
	}

	// Fall back to the namespace associated with the service account token, if available
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return "default"
}

func main() {
	namespace := Namespace()
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := metadata.NewForConfig(metadata.ConfigFor(config)) //I am not sure if ConfigFor must be used
	if err != nil {
		panic(err.Error())
	}
	// creates a factory for a metadatainformer
	factory := metadatainformer.NewFilteredSharedInformerFactory(clientset,
		0,
		namespace,
		nil)
	// we want to observe deployments
	gvr, _ := schema.ParseResourceArg("deployments.v1.apps")
	i := factory.ForResource(*gvr)

	//List all objects (deployments) for testing purposes
	allobj, _ := i.Lister().List(labels.Everything())
	fmt.Println("Debug: Len of allobj: ", len(allobj))
	for _, obj := range allobj {
		newObj := obj.(*metav1.PartialObjectMetadata)
		deploymentMetaNew := newObj.ObjectMeta
		fmt.Println("Running Loop... ")
		fmt.Println("Debug: Metadata for " + deploymentMetaNew.Name)
		fmt.Println(" Annotations:")
		for annotation, value := range deploymentMetaNew.GetAnnotations() {
			fmt.Println(annotation + "=" + value)
		}
	}

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(newObj interface{}) {
			deploymentPMetaNew := newObj.(*metav1.PartialObjectMetadata)
			deploymentMetaNew := deploymentPMetaNew.ObjectMeta
			fmt.Println("Debug: Metadata added " + deploymentMetaNew.Name)
			fmt.Println("Annotations:")
			for annotation, value := range deploymentMetaNew.GetAnnotations() {
				fmt.Println(annotation + "=" + value)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			deploymentPMetaOld := oldObj.(*metav1.PartialObjectMetadata)
			deploymentPMetaNew := newObj.(*metav1.PartialObjectMetadata)
			deploymentMetaOld := deploymentPMetaOld.ObjectMeta
			deploymentMetaNew := deploymentPMetaNew.ObjectMeta
			fmt.Println("Debug: Metadata changed " + deploymentMetaOld.Name)
			fmt.Println("Annotations:")
			for annotation, value := range deploymentMetaNew.GetAnnotations() {
				fmt.Println(annotation + "=" + value)
			}
		},
	}

	stopper := make(chan struct{})
	informer := i.Informer()
	informer.AddEventHandler(handler)
	factory.Start(stopper)

	runtime.Goexit()
}
