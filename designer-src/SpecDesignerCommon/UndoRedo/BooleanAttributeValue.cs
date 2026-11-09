using System;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000030 RID: 48
	public class BooleanAttributeValue
	{
		// Token: 0x17000058 RID: 88
		// (get) Token: 0x06000171 RID: 369 RVA: 0x0000791E File Offset: 0x00005B1E
		// (set) Token: 0x06000172 RID: 370 RVA: 0x00007926 File Offset: 0x00005B26
		public bool OldValue { get; set; }

		// Token: 0x17000059 RID: 89
		// (get) Token: 0x06000173 RID: 371 RVA: 0x0000792F File Offset: 0x00005B2F
		// (set) Token: 0x06000174 RID: 372 RVA: 0x00007937 File Offset: 0x00005B37
		public bool NewValue { get; set; }
	}
}
