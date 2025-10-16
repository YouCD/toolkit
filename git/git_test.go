package git

import (
	"fmt"
	"testing"

	"github.com/klauspost/cpuid/v2"
)

func TestGit_PlainClone(t *testing.T) {
	cpu := cpuid.CPU
	/*
	 vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ss syscall nx rdtscp lm constant_tsc arch_perfmon nopl xto
	                                 pology tsc_reliable nonstop_tsc cpuid pni pclmulqdq ssse3 cx16 pcid sse4_1 sse4_2 x2apic popcnt tsc_deadline_timer aes xsave avx f16c rdrand hypervisor lahf
	                                 _lm cpuid_fault pti ssbd ibrs ibpb stibp fsgsbase tsc_adjust smep arat md_clear flush_l1d arch_capabilities

	*/
	//cpu.Supports(cpuid.X87, cpuid.NRIPS, cpuid.VME, cpuid.DE, cpuid.PSE, cpuid.TSC, cpuid.MSR, cpuid.PAE, cpuid.MCE, cpuid.CX8, cpuid.APIC, cpuid.SEP, cpuid.MTRR, cpuid.PGE, cpuid.MCA, cpuid.CMOV, cp)
	fmt.Println(cpu.FeatureSet())
	fmt.Println(len(cpu.FeatureSet()))
}
