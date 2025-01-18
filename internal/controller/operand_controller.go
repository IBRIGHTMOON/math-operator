/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batchv1 "github.com/IBRIGHTMOON/math-operator/api/v1"
)

// OperandReconciler reconciles a Operand object
type OperandReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch.math.operator.com,resources=operands,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.math.operator.com,resources=operands/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.math.operator.com,resources=operands/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Operand object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *OperandReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// TODO(user): your logic here
	logger.Info("Reconciling Operand")

	// fetching operator
	logger.Info("Fetching Operator")
	operator := &batchv1.Operator{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: "default"}, operator); err != nil {
		logger.Error(err, "unable to fetch Operator")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("Successful fetched Operator")

	// Fetching Operand
	logger.Info("Fetching Operand")
	operand := &batchv1.Operand{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, operand); err != nil {
		logger.Error(err, "unable to fetch Operand")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("Successful fetched Operand")

	// Fetching Result
	logger.Info("Fetching Result")
	result := &batchv1.Result{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, result); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Result not found, creating a new one")

			result = &batchv1.Result{
				ObjectMeta: metav1.ObjectMeta{
					Name:      req.Name,
					Namespace: req.Namespace,
				},
				Status: batchv1.ResultStatus{
					Result:      0,
					Calculation: "",
				},
			}

			if err := r.Client.Create(ctx, result); err != nil {
				logger.Error(err, "unable to create Result")
				return ctrl.Result{}, err
			}
			logger.Info("Successful created Result")
		} else {
			logger.Error(err, "unable to fetch Result")
			return ctrl.Result{}, err
		}
	}
	logger.Info("Successful fetched Result")

	// If Clear is true, set the result to operator.Value; otherwise calculate the new result with respect to the operator
	if operand.Spec.Clear {
		if val, err := Calculate(operator.Spec.Operation, operand.Spec.Value, 0); err != nil {
			logger.Error(err, "unable to calculate result")
			return ctrl.Result{}, err
		} else {
			result.Status.Result = val
		}
		result.Status.Calculation = fmt.Sprintf("%d", operand.Spec.Value)
	} else {
		if val, err := Calculate(operator.Spec.Operation, operand.Spec.Value, result.Status.Result); err != nil {
			logger.Error(err, "unable to calculate result")
			return ctrl.Result{}, err
		} else {
			result.Status.Result = val

			if calculation, err := GenerateCalculationString(result.Status.Calculation, operator.Spec.Operation, operand.Spec.Value); err != nil {
				logger.Error(err, "unable to generate calculation string")
				return ctrl.Result{}, err
			} else {
				result.Status.Calculation = calculation
			}
		}
	}

	logger.Info("Updating Result")
	if err := r.Status().Update(ctx, result); err != nil {
		logger.Error(err, "unable to update Result")
		return ctrl.Result{}, err
	}
	logger.Info("Succesfuly updated Result")
	logger.Info(fmt.Sprintf("New Result is: %d", result.Status.Result))

	logger.Info("Reconciliation ended for Operand")

	return ctrl.Result{}, nil
}

func GenerateCalculationString(prevCalculation string, operation batchv1.Operation, value int64) (string, error) {
	switch operation {
	case batchv1.Addition:
		return fmt.Sprintf("%s + %d", prevCalculation, value), nil
	case batchv1.Subtraction:
		return fmt.Sprintf("%s - %d", prevCalculation, value), nil
	case batchv1.Multiplication:
		return fmt.Sprintf("(%s) * %d", prevCalculation, value), nil
	case batchv1.Division:
		return fmt.Sprintf("(%s) / %d", prevCalculation, value), nil
	}

	return "", errors.NewBadRequest("Operation is invalid")
}

func Calculate(operation batchv1.Operation, value int64, oldResult int64) (int64, error) {
	switch operation {
	case batchv1.Addition:
		return oldResult + value, nil
	case batchv1.Subtraction:
		return oldResult - value, nil
	case batchv1.Multiplication:
		return oldResult * value, nil
	case batchv1.Division:
		if value == 0 {
			return -1, errors.NewBadRequest("Division by zero")
		}
		return oldResult / value, nil
	default:
		return 0, errors.NewBadRequest("Unknown operation")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperandReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Operand{}).
		Named("operand").
		Complete(r)
}
