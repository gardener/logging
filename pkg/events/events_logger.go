package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	kubeinformers "k8s.io/client-go/informers"
	kubeinformersinterfaces "k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func GetController(clientset *kubernetes.Clientset, namespace string, fieldSelector fields.Selector) cache.Controller {
	watchlist := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"events",
		namespace,
		fieldSelector,
	)
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Event{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				j, _ := json.Marshal(obj)
				fmt.Printf("%s\n", string(j))
			},
		},
	)
	return controller
}

type EventWatcherConfig struct {
	Kubeconfig string //Do I need this field?
	Namespace  string
}

type GardenerEventWatcherConfig struct {
	SeedEventWatcherConfig   EventWatcherConfig
	SeedKubeInformerFactory  kubeinformers.SharedInformerFactory
	ShootEventWatcherConfig  EventWatcherConfig
	ShootKubeInformerFactory kubeinformers.SharedInformerFactory
}

func (e *GardenerEventWatcherConfig) New() *GardenerEventWatcher {
	_ = e.SeedKubeInformerFactory.InformerFor(&v1.Event{},
		NewEventInformerFuncForNamespace(
			"Seed",
			e.SeedEventWatcherConfig.Namespace,
		),
	)

	_ = e.ShootKubeInformerFactory.InformerFor(&v1.Event{},
		NewEventInformerFuncForNamespace(
			"Shoot",
			e.ShootEventWatcherConfig.Namespace,
		),
	)

	return &GardenerEventWatcher{
		SeedKubeInformerFactory:  e.SeedKubeInformerFactory,
		ShootKubeInformerFactory: e.ShootKubeInformerFactory,
	}
}

type GardenerEventWatcher struct {
	SeedKubeInformerFactory  kubeinformers.SharedInformerFactory
	ShootKubeInformerFactory kubeinformers.SharedInformerFactory
}

func (e *GardenerEventWatcher) Run(stopCh <-chan struct{}) {
	e.SeedKubeInformerFactory.Start(stopCh)
	e.ShootKubeInformerFactory.Start(stopCh)
	<-stopCh
}

///////////////
// Options has all the context and parameters needed to run a Gardener Event Logger.
type Options struct {
	Kubeconfig string
	Namespace  string
}

func (o *Options) Validate() []error {
	//TODO: vlvasilev implement me
	errors := []error{}
	errors = append(errors, nil)
	return errors
}

func (o *Options) ApplyTo(config *EventWatcherConfig) error {
	config.Kubeconfig = o.Kubeconfig
	config.Namespace = o.Namespace
	return nil
}

type SeedOptions struct {
	Options
}

// AddFlags adds all flags to the given FlagSet.
func (o *SeedOptions) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.StringVar(&o.Kubeconfig, "seed-kubeconfig", "", "The kubeconfig for the seed cluster")
	fs.StringVar(&o.Namespace, "seed-event-namespace", "kube-system", "The namespace of the seed events")
}

// Validate all flags of the given Options.
func (o *SeedOptions) Validate() []error {
	return o.Options.Validate()
}

func (o *SeedOptions) ApplyTo(config *EventWatcherConfig) error {
	return o.Options.ApplyTo(config)
}

type ShootOptions struct {
	Options
}

// AddFlags adds all flags to the given FlagSet.
func (o *ShootOptions) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.StringVar(&o.Kubeconfig, "shoot-kubeconfig", "", "The kubeconfig for the shoot cluster")
	fs.StringVar(&o.Namespace, "shoot-event-namespace", "kube-system", "The namespace of the shoot events")
}

// Validate all flags of the given Options.
func (o *ShootOptions) Validate() []error {
	return o.Options.Validate()
}

func (o *ShootOptions) ApplyTo(config *EventWatcherConfig) error {
	return o.Options.ApplyTo(config)
}

func NewEventInformerFuncForNamespace(origin, namespace string) kubeinformersinterfaces.NewInformerFunc {
	return func(clientset kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
		watchlist := cache.NewListWatchFromClient(
			clientset.CoreV1().RESTClient(),
			"events",
			namespace,
			fields.Everything(),
		)
		informer := cache.NewSharedIndexInformer(
			watchlist,
			&v1.Event{},
			resyncPeriod,
			cache.Indexers{},
		)
		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				event, ok := getCorev1Event(obj, origin)
				if !ok {
					return
				}
				j, _ := json.Marshal(event)
				fmt.Printf("%s\n", string(j))
			},
		})
		return informer
	}
}

type event struct {
	Origin         string      `json:"origin" protobuf:"bytes,9,name=origin"`
	Type           string      `json:"type,omitempty" protobuf:"bytes,9,opt,name=type"`
	Count          int32       `json:"count,omitempty" protobuf:"varint,8,opt,name=count"`
	FirstTimestamp metav1.Time `json:"firstTimestamp,omitempty" protobuf:"bytes,6,opt,name=firstTimestamp"`
	LastTimestamp  metav1.Time `json:"lastTimestamp,omitempty" protobuf:"bytes,7,opt,name=lastTimestamp"`
	Reason         string      `json:"reason,omitempty" protobuf:"bytes,3,opt,name=reason"`
	Object         string      `json:"object" protobuf:"bytes,2,opt,name=object"`
	Message        string      `json:"_entry,omitempty" protobuf:"bytes,4,opt,name=_entry"`
	Source         string      `json:"source,omitempty" protobuf:"bytes,5,opt,name=source"`
	SourceHost     string      `json:"sourceHost,omitempty" protobuf:"bytes,2,opt,name=sourceHost"`
}

func getCorev1Event(obj interface{}, origin string) (*event, bool) {
	eventObj, ok := obj.(*v1.Event)
	if !ok {
		return nil, false
	}

	involvedObject := eventObj.InvolvedObject.Name
	if eventObj.InvolvedObject.Kind != "" {
		involvedObject = eventObj.InvolvedObject.Kind + "/" + involvedObject
	}

	return &event{
		Origin:         origin,
		Type:           eventObj.Type,
		Count:          eventObj.Count,
		FirstTimestamp: eventObj.FirstTimestamp,
		LastTimestamp:  eventObj.LastTimestamp,
		Reason:         eventObj.Reason,
		Object:         involvedObject,
		Message:        eventObj.Message,
		Source:         eventObj.Source.Component,
		SourceHost:     eventObj.Source.Host,
	}, true
}
