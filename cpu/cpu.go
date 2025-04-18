package main

import (
	. "github.com/klauspost/cpuid/v2"
)

func main() {

	CPU.Supports(AESNI, AVX, AVX2, AVX512F, AVX512BW, AVX512CD, AVX512VL, AVX512DQ, AVX512VBMI, AVX512VBMI2, AVX512VNNI, AVX512BITALG, AVX512VPOPCNTDQ, AVX512VP2INTERSECT, AVX512VL)
	//supportsVMX := cpuid.CPU.Supports(cpuid.VMX) // Intel VT-x
	//supportsSVM := cpuid.CPU.Supports(cpuid.SVM) // AMD-V
	//if supportsVMX || supportsSVM {
	//	fmt.Println("✅ CPU supports hardware virtualization for KVM.")
	//} else {
	//	fmt.Println("❌ CPU does NOT support required virtualization extensions for KVM.")
	//}

}
