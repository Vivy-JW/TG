using System;
using SpecDesignerCommon;

namespace SpecDesigner.FormEditor.Events
{
	// Token: 0x02000017 RID: 23
	public class Parse4fdEventArgs
	{
		// Token: 0x1700002C RID: 44
		// (get) Token: 0x060000CA RID: 202 RVA: 0x00005209 File Offset: 0x00003409
		// (set) Token: 0x060000CB RID: 203 RVA: 0x00005211 File Offset: 0x00003411
		public PackageKey ProgramKey { get; set; }

		// Token: 0x1700002D RID: 45
		// (get) Token: 0x060000CC RID: 204 RVA: 0x0000521A File Offset: 0x0000341A
		// (set) Token: 0x060000CD RID: 205 RVA: 0x00005222 File Offset: 0x00003422
		public string FormContent { get; set; }
	}
}
