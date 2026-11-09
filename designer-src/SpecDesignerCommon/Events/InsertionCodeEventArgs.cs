using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x0200001E RID: 30
	public class InsertionCodeEventArgs
	{
		// Token: 0x17000026 RID: 38
		// (get) Token: 0x060000DB RID: 219 RVA: 0x00005D1E File Offset: 0x00003F1E
		// (set) Token: 0x060000DC RID: 220 RVA: 0x00005D26 File Offset: 0x00003F26
		public PackageKey ProgramKey { get; set; }

		// Token: 0x17000027 RID: 39
		// (get) Token: 0x060000DD RID: 221 RVA: 0x00005D2F File Offset: 0x00003F2F
		// (set) Token: 0x060000DE RID: 222 RVA: 0x00005D37 File Offset: 0x00003F37
		public string Content { get; set; }
	}
}
