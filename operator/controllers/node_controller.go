/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/jaberchez/k8s-operator-dns-ops/helpers"
	"github.com/jaberchez/k8s-operator-dns-ops/powerdns"
)

// NodeReconciler reconciles a Node object
type NodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=nodes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Node object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logr := log.FromContext(ctx)

	var dnsActions helpers.DNSActions
	dnsType := os.Getenv("DNS_TYPE")

	switch strings.ToLower(dnsType) {
	case "powerdns":
		p := &powerdns.PowerDNS{
			Server: os.Getenv("POWERDNS_SERVER"),
			Key:    os.Getenv("POWERDNS_KEY"),
		}

		dnsActions = p
	}

	node := &corev1.Node{}

	if err := r.Get(ctx, req.NamespacedName, node); err != nil {
		if apierrors.IsNotFound(err) {
			// We'll ignore not-found errors, since we can get them on deleted requests.
			return ctrl.Result{}, nil
		}

		logr.Error(err, "unable to fetch Node")
		return ctrl.Result{}, err
	}

	nodeFinalizerName := "node.example.com/finalizer"
	nodeName := node.ObjectMeta.Name
	dnsRecord := helpers.DNSRecord{}

	// Get the current IP's asigned to this node (status.addresses[])
	if node.Status.Addresses != nil && len(node.Status.Addresses) == 0 {
		return ctrl.Result{}, fmt.Errorf("ip addresses not found in node %s", nodeName)
	}

	var nodeIps []string

	for i := range node.Status.Addresses {
		a := node.Status.Addresses[i]

		if a.Type == corev1.NodeInternalIP || a.Type == corev1.NodeInternalIP {
			nodeIps = append(nodeIps, a.Address)
		}
	}

	// From nodeName get the DNS zone (the last part from FQDN)
	dnsZone, err := helpers.GetDnsZone(nodeName)

	if err != nil {
		return ctrl.Result{}, err
	}

	// Get DNS A records for this node
	if dnsRecord = dnsActions.GetARecord(nodeName, dnsZone); dnsRecord.Err != nil {
		return ctrl.Result{}, dnsRecord.Err
	}

	// Examine DeletionTimestamp to determine if object is under deletion
	if !node.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		//
		// Remove the DNS Record
		if err = dnsActions.DeleteARecord(nodeName, dnsZone); err != nil {
			return ctrl.Result{}, err
		}

		// Remove the finalizer
		if containsString(node.GetFinalizers(), nodeFinalizerName) {
			// Remove our finalizer from the list and update it
			controllerutil.RemoveFinalizer(node, nodeFinalizerName)

			if err := r.Update(ctx, node); err != nil {
				return ctrl.Result{}, err
			}
		}

		// OK, stop reconciliation as the item is being deleted and the
		// DNS was removed
		return ctrl.Result{}, nil
	} else {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(node.GetFinalizers(), nodeFinalizerName) {
			controllerutil.AddFinalizer(node, nodeFinalizerName)

			if err := r.Update(ctx, node); err != nil {
				if apierrors.IsConflict(err) {
					// The Node has been updated since we read it.
					// Requeue the Node to try to reconciliate again.
					return ctrl.Result{Requeue: true}, nil
				}

				if apierrors.IsNotFound(err) {
					// The Node has been deleted since we read it.
					// Requeue the Node to try to reconciliate again.
					return ctrl.Result{Requeue: true}, nil
				}

				logr.Error(err, "unable to update Node")
				return ctrl.Result{}, err
			}
		}

		// Check if record exits
		if dnsRecord.Found {
			// DNS record exists, check IPs
			if !reflect.DeepEqual(dnsRecord.Records, nodeIps) {
				// Update DNS record
				if err = dnsActions.UpdateARecord(nodeName, dnsZone, nodeIps); err != nil {
					return ctrl.Result{}, err
				}
			}
		} else {
			// DNS record not found, add it
			if err = dnsActions.CreateARecord(nodeName, dnsZone, nodeIps); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(r)
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
