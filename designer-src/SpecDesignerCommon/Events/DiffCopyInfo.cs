using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x0200001F RID: 31
	public class DiffCopyInfo
	{
		// Token: 0x17000028 RID: 40
		// (get) Token: 0x060000E1 RID: 225 RVA: 0x00005D50 File Offset: 0x00003F50
		// (set) Token: 0x060000E2 RID: 226 RVA: 0x00005D58 File Offset: 0x00003F58
		public PackageKey ProgramKey { get; set; }

		// Token: 0x17000029 RID: 41
		// (get) Token: 0x060000E3 RID: 227 RVA: 0x00005D61 File Offset: 0x00003F61
		// (set) Token: 0x060000E4 RID: 228 RVA: 0x00005D69 File Offset: 0x00003F69
		public string SelectContent { get; set; }

		// Token: 0x1700002A RID: 42
		// (get) Token: 0x060000E5 RID: 229 RVA: 0x00005D72 File Offset: 0x00003F72
		// (set) Token: 0x060000E6 RID: 230 RVA: 0x00005D7A File Offset: 0x00003F7A
		public int startLine { get; set; }

		// Token: 0x1700002B RID: 43
		// (get) Token: 0x060000E7 RID: 231 RVA: 0x00005D83 File Offset: 0x00003F83
		// (set) Token: 0x060000E8 RID: 232 RVA: 0x00005D8B File Offset: 0x00003F8B
		public int endLine { get; set; }

		// Token: 0x1700002C RID: 44
		// (get) Token: 0x060000E9 RID: 233 RVA: 0x00005D94 File Offset: 0x00003F94
		// (set) Token: 0x060000EA RID: 234 RVA: 0x00005D9C File Offset: 0x00003F9C
		public int lineNumber { get; set; }

		// Token: 0x1700002D RID: 45
		// (get) Token: 0x060000EB RID: 235 RVA: 0x00005DA5 File Offset: 0x00003FA5
		// (set) Token: 0x060000EC RID: 236 RVA: 0x00005DAD File Offset: 0x00003FAD
		public bool IsDiffBaseOnStandard { get; set; }

		// Token: 0x1700002E RID: 46
		// (get) Token: 0x060000ED RID: 237 RVA: 0x00005DB6 File Offset: 0x00003FB6
		// (set) Token: 0x060000EE RID: 238 RVA: 0x00005DBE File Offset: 0x00003FBE
		public string FunctionName { get; set; }
	}
}
