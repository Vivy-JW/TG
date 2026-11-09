using System;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000025 RID: 37
	[Flags]
	public enum Status
	{
		// Token: 0x04000052 RID: 82
		NULL = 0,
		// Token: 0x04000053 RID: 83
		CREATE = 2,
		// Token: 0x04000054 RID: 84
		DELETE = 4,
		// Token: 0x04000055 RID: 85
		MODIFY = 8
	}
}
