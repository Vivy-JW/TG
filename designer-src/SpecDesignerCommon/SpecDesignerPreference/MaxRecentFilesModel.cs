using System;
using System.Collections.Generic;

namespace SpecDesignerPreference
{
	// Token: 0x0200008A RID: 138
	public sealed class MaxRecentFilesModel
	{
		// Token: 0x170001A9 RID: 425
		// (get) Token: 0x060005BA RID: 1466 RVA: 0x0001A4DD File Offset: 0x000186DD
		// (set) Token: 0x060005BB RID: 1467 RVA: 0x0001A4E5 File Offset: 0x000186E5
		public int MaxKey { get; set; }

		// Token: 0x170001AA RID: 426
		// (get) Token: 0x060005BC RID: 1468 RVA: 0x0001A4EE File Offset: 0x000186EE
		// (set) Token: 0x060005BD RID: 1469 RVA: 0x0001A4F6 File Offset: 0x000186F6
		public string MaxValue { get; set; }

		// Token: 0x060005BF RID: 1471 RVA: 0x0001A508 File Offset: 0x00018708
		public static List<MaxRecentFilesModel> Load()
		{
			return new List<MaxRecentFilesModel>
			{
				new MaxRecentFilesModel
				{
					MaxKey = 10,
					MaxValue = "10"
				},
				new MaxRecentFilesModel
				{
					MaxKey = 20,
					MaxValue = "20"
				},
				new MaxRecentFilesModel
				{
					MaxKey = 30,
					MaxValue = "30"
				}
			};
		}
	}
}
