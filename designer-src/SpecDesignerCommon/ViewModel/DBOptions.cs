using System;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000A8 RID: 168
	public class DBOptions
	{
		// Token: 0x17000206 RID: 518
		// (get) Token: 0x0600071B RID: 1819 RVA: 0x0001FBF1 File Offset: 0x0001DDF1
		// (set) Token: 0x0600071C RID: 1820 RVA: 0x0001FBF9 File Offset: 0x0001DDF9
		public int Value { get; set; }

		// Token: 0x17000207 RID: 519
		// (get) Token: 0x0600071D RID: 1821 RVA: 0x0001FC02 File Offset: 0x0001DE02
		// (set) Token: 0x0600071E RID: 1822 RVA: 0x0001FC0A File Offset: 0x0001DE0A
		public string Description { get; set; }

		// Token: 0x0600071F RID: 1823 RVA: 0x0001FC14 File Offset: 0x0001DE14
		public DBOptions(string value, string description)
		{
			int num = 0;
			if (int.TryParse(value, out num))
			{
				this.Value = num;
				this.Description = description;
				return;
			}
			throw new ArgumentException("For_DB");
		}
	}
}
