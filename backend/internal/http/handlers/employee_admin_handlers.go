package handlers

import (
	"context"
	"net/http"

	"syslog/internal/domain"
	"syslog/internal/service"
)

type EmployeeAdminWriter interface {
	CreateEmployee(context.Context, service.EmployeeWriteInput) (*domain.Employee, error)
	UpdateEmployee(context.Context, uint64, service.EmployeeWriteInput) (*domain.Employee, error)
	DisableEmployee(context.Context, uint64) (*domain.Employee, error)
}

type employeeWriteRequest struct {
	EmployeeNo string                  `json:"employeeNo"`
	SystemNo   string                  `json:"systemNo"`
	Name       string                  `json:"name"`
	Status     string                  `json:"status"`
	Devices    []employeeDeviceRequest `json:"devices"`
}

type employeeDeviceRequest struct {
	MacAddress  string `json:"macAddress"`
	DeviceLabel string `json:"deviceLabel"`
	Status      string `json:"status"`
}

type employeeWriteResponse struct {
	Employee domain.Employee `json:"employee"`
}

func NewEmployeeCreateHandler(admin EmployeeAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		var req employeeWriteRequest
		if err := decodeJSONBody(r, &req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		employee, err := admin.CreateEmployee(r.Context(), employeeWriteInput(req))
		if err != nil {
			http.Error(w, http.StatusText(statusCodeForServiceError(err)), statusCodeForServiceError(err))
			return
		}

		writeJSON(w, http.StatusCreated, employeeWriteResponse{Employee: *employee})
	}
}

func NewEmployeeUpdateHandler(admin EmployeeAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		id, err := parseUint64PathValue(r, "id")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var req employeeWriteRequest
		if err := decodeJSONBody(r, &req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		employee, err := admin.UpdateEmployee(r.Context(), id, employeeWriteInput(req))
		if err != nil {
			http.Error(w, http.StatusText(statusCodeForServiceError(err)), statusCodeForServiceError(err))
			return
		}

		writeJSON(w, http.StatusOK, employeeWriteResponse{Employee: *employee})
	}
}

func NewEmployeeDisableHandler(admin EmployeeAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		id, err := parseUint64PathValue(r, "id")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		employee, err := admin.DisableEmployee(r.Context(), id)
		if err != nil {
			http.Error(w, http.StatusText(statusCodeForServiceError(err)), statusCodeForServiceError(err))
			return
		}

		writeJSON(w, http.StatusOK, employeeWriteResponse{Employee: *employee})
	}
}

func employeeWriteInput(req employeeWriteRequest) service.EmployeeWriteInput {
	devices := make([]service.EmployeeDeviceInput, 0, len(req.Devices))
	for _, device := range req.Devices {
		devices = append(devices, service.EmployeeDeviceInput{
			MacAddress:  device.MacAddress,
			DeviceLabel: device.DeviceLabel,
			Status:      device.Status,
		})
	}

	return service.EmployeeWriteInput{
		EmployeeNo: req.EmployeeNo,
		SystemNo:   req.SystemNo,
		Name:       req.Name,
		Status:     req.Status,
		Devices:    devices,
	}
}
