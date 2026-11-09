using System;
using SpecDesignerCommon;

namespace SpecDesigner.Infrastructure.Event
{
	// Token: 0x02000007 RID: 7
	public class ModifyInfomation
	{
		// Token: 0x17000003 RID: 3
		// (get) Token: 0x06000010 RID: 16 RVA: 0x000021DF File Offset: 0x000003DF
		// (set) Token: 0x06000011 RID: 17 RVA: 0x000021E7 File Offset: 0x000003E7
		public PackageKey ProgramKey { get; set; }

		// Token: 0x17000004 RID: 4
		// (get) Token: 0x06000012 RID: 18 RVA: 0x000021F0 File Offset: 0x000003F0
		// (set) Token: 0x06000013 RID: 19 RVA: 0x000021F8 File Offset: 0x000003F8
		public string Source { get; set; }

		// Token: 0x17000005 RID: 5
		// (get) Token: 0x06000014 RID: 20 RVA: 0x00002201 File Offset: 0x00000401
		// (set) Token: 0x06000015 RID: 21 RVA: 0x00002209 File Offset: 0x00000409
		public string Modified { get; set; }

		// Token: 0x17000006 RID: 6
		// (get) Token: 0x06000016 RID: 22 RVA: 0x00002212 File Offset: 0x00000412
		// (set) Token: 0x06000017 RID: 23 RVA: 0x0000221A File Offset: 0x0000041A
		public ModifyInfomation.ModifyTypeEnum ModifyType { get; set; }

		// Token: 0x02000008 RID: 8
		public enum ModifyTypeEnum
		{
			// Token: 0x04000009 RID: 9
			Name = 2,
			// Token: 0x0400000A RID: 10
			Other = 4,
			// Token: 0x0400000B RID: 11
			Both = 6
		}
	}
}
