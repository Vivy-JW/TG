using System;
using System.Collections.Generic;

namespace SpecDesignerPreference
{
	// Token: 0x0200008B RID: 139
	public class RecentFilesModel
	{
		// Token: 0x060005C0 RID: 1472 RVA: 0x0001A582 File Offset: 0x00018782
		public RecentFilesModel()
		{
			this.results = new List<string>();
		}

		// Token: 0x170001AB RID: 427
		// (get) Token: 0x060005C1 RID: 1473 RVA: 0x0001A595 File Offset: 0x00018795
		// (set) Token: 0x060005C2 RID: 1474 RVA: 0x0001A59D File Offset: 0x0001879D
		public List<string> RecentSpecList
		{
			get
			{
				return this.results;
			}
			set
			{
				this.results.Add(this.sFileName);
			}
		}

		// Token: 0x04000233 RID: 563
		private string sFileName;

		// Token: 0x04000234 RID: 564
		private List<string> results;
	}
}
