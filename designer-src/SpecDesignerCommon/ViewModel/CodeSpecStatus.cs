using System;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000FC RID: 252
	[Flags]
	public enum CodeSpecStatus
	{
		// Token: 0x040002E5 RID: 741
		NULL = 0,
		// Token: 0x040002E6 RID: 742
		LOADED = 1,
		// Token: 0x040002E7 RID: 743
		CREATE = 2,
		// Token: 0x040002E8 RID: 744
		DELETE = 4,
		// Token: 0x040002E9 RID: 745
		MODIFY = 8
	}
}
