using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x02000020 RID: 32
	public class FunctionSyncInfo
	{
		// Token: 0x1700002F RID: 47
		// (get) Token: 0x060000F0 RID: 240 RVA: 0x00005DCF File Offset: 0x00003FCF
		// (set) Token: 0x060000F1 RID: 241 RVA: 0x00005DD7 File Offset: 0x00003FD7
		public PackageKey ProgramKey { get; set; }

		// Token: 0x17000030 RID: 48
		// (get) Token: 0x060000F2 RID: 242 RVA: 0x00005DE0 File Offset: 0x00003FE0
		// (set) Token: 0x060000F3 RID: 243 RVA: 0x00005DE8 File Offset: 0x00003FE8
		public string Content { get; set; }

		// Token: 0x17000031 RID: 49
		// (get) Token: 0x060000F4 RID: 244 RVA: 0x00005DF1 File Offset: 0x00003FF1
		// (set) Token: 0x060000F5 RID: 245 RVA: 0x00005DF9 File Offset: 0x00003FF9
		public string FunctionName { get; set; }

		// Token: 0x17000032 RID: 50
		// (get) Token: 0x060000F6 RID: 246 RVA: 0x00005E02 File Offset: 0x00004002
		// (set) Token: 0x060000F7 RID: 247 RVA: 0x00005E0A File Offset: 0x0000400A
		public string parameter { get; set; }

		// Token: 0x17000033 RID: 51
		// (get) Token: 0x060000F8 RID: 248 RVA: 0x00005E13 File Offset: 0x00004013
		// (set) Token: 0x060000F9 RID: 249 RVA: 0x00005E1B File Offset: 0x0000401B
		public string FullFunction { get; set; }

		// Token: 0x17000034 RID: 52
		// (get) Token: 0x060000FA RID: 250 RVA: 0x00005E24 File Offset: 0x00004024
		// (set) Token: 0x060000FB RID: 251 RVA: 0x00005E2C File Offset: 0x0000402C
		public int SortIndex { get; set; }

		// Token: 0x17000035 RID: 53
		// (get) Token: 0x060000FC RID: 252 RVA: 0x00005E35 File Offset: 0x00004035
		// (set) Token: 0x060000FD RID: 253 RVA: 0x00005E3D File Offset: 0x0000403D
		public bool TapSrcIsNull { get; set; }
	}
}
