using System;
using System.ComponentModel;

namespace SpecDesignerCommon
{
	// Token: 0x0200000E RID: 14
	public enum TzpType
	{
		// Token: 0x04000028 RID: 40
		None,
		// Token: 0x04000029 RID: 41
		[Description("4RP")]
		Report,
		// Token: 0x0400002A RID: 42
		[Description("RSPEC")]
		ReportSpec,
		// Token: 0x0400002B RID: 43
		[Description("GCODE")]
		ReportCode,
		// Token: 0x0400002C RID: 44
		[Description("CODE")]
		Code,
		// Token: 0x0400002D RID: 45
		[Description("CSPEC")]
		CodeSpec,
		// Token: 0x0400002E RID: 46
		[Description("SPEC")]
		Form,
		// Token: 0x0400002F RID: 47
		Service
	}
}
