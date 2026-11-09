using System;

namespace SpecDesignerCommon
{
	// Token: 0x02000138 RID: 312
	public class FieldArgs
	{
		// Token: 0x170002DE RID: 734
		// (get) Token: 0x06000AF8 RID: 2808 RVA: 0x00035671 File Offset: 0x00033871
		// (set) Token: 0x06000AF9 RID: 2809 RVA: 0x00035679 File Offset: 0x00033879
		public string Field { get; set; }

		// Token: 0x170002DF RID: 735
		// (get) Token: 0x06000AFA RID: 2810 RVA: 0x00035682 File Offset: 0x00033882
		// (set) Token: 0x06000AFB RID: 2811 RVA: 0x0003568A File Offset: 0x0003388A
		public PackageKey ProgramKey { get; set; }

		// Token: 0x06000AFC RID: 2812 RVA: 0x00035693 File Offset: 0x00033893
		public FieldArgs(string field, PackageKey key)
		{
			this.Field = field;
			this.ProgramKey = key;
		}
	}
}
