package cl

// #include <CL/opencl.h>
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

type BuildError string

func (e BuildError) Error() string {
	return fmt.Sprintf("cl: build error (%s)", string(e))
}

type Program struct {
	clProgram C.cl_program
	devices   []*Device
}

func releaseProgram(p *Program) {
	if p.clProgram != nil {
		C.clReleaseProgram(p.clProgram)
		p.clProgram = nil
	}
}

func (p *Program) Release() {
	releaseProgram(p)
}

func (p *Program) BuildProgram(devices []*Device, options string) error {
	var cOptions *C.char
	if options != "" {
		cOptions = C.CString(options)
		defer C.free(unsafe.Pointer(cOptions))
	}
	var deviceList []C.cl_device_id
	var deviceListPtr *C.cl_device_id
	if devices != nil && len(devices) > 0 {
		deviceList = buildDeviceIdList(devices)
		deviceListPtr = &deviceList[0]
	}
	numDevices := C.cl_uint(len(deviceList))

	if err := C.clBuildProgram(p.clProgram, numDevices, deviceListPtr, cOptions, nil, nil); err != C.CL_SUCCESS {
		if err != C.CL_BUILD_PROGRAM_FAILURE {
			return toError(err)
		}
		var bLen C.size_t
		var err C.cl_int
		err = C.clGetProgramBuildInfo(p.clProgram, p.devices[0].id, C.CL_PROGRAM_BUILD_LOG, 0, nil, &bLen)
		if err != C.CL_SUCCESS {
			return toError(err)
		}
		buffer := make([]byte, bLen)
		err = C.clGetProgramBuildInfo(p.clProgram, p.devices[0].id, C.CL_PROGRAM_BUILD_LOG, C.size_t(len(buffer)), unsafe.Pointer(&buffer[0]), &bLen)
		if err != C.CL_SUCCESS {
			return toError(err)
		}
		return BuildError(string(buffer[:bLen-1]))
	}
	return nil
}

func (p *Program) CreateKernel(name string) (*Kernel, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	var err C.cl_int
	clKernel := C.clCreateKernel(p.clProgram, cName, &err)
	if err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	kernel := &Kernel{clKernel: clKernel, name: name}
	runtime.SetFinalizer(kernel, releaseKernel)
	return kernel, nil
}
