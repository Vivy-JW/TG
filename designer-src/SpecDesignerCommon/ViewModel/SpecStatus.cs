using System;
using System.ComponentModel;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000040 RID: 64
	[Flags]
	public enum SpecStatus
	{
		// Token: 0x040000C3 RID: 195
		NULL = 0,
		// Token: 0x040000C4 RID: 196
		[Description("c")]
		CREATE = 2,
		// Token: 0x040000C5 RID: 197
		[Description("d")]
		DELETE = 4,
		// Token: 0x040000C6 RID: 198
		[Description("u")]
		MODIFY = 8
	}
}
