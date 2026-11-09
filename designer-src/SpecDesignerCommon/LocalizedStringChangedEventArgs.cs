using System;

namespace SpecDesignerCommon
{
	// Token: 0x0200005E RID: 94
	public class LocalizedStringChangedEventArgs
	{
		// Token: 0x170000BF RID: 191
		// (get) Token: 0x06000324 RID: 804 RVA: 0x0000CE4A File Offset: 0x0000B04A
		// (set) Token: 0x06000325 RID: 805 RVA: 0x0000CE52 File Offset: 0x0000B052
		public string Name { get; set; }

		// Token: 0x170000C0 RID: 192
		// (get) Token: 0x06000326 RID: 806 RVA: 0x0000CE5B File Offset: 0x0000B05B
		// (set) Token: 0x06000327 RID: 807 RVA: 0x0000CE63 File Offset: 0x0000B063
		public string LocalizedString { get; set; }

		// Token: 0x170000C1 RID: 193
		// (get) Token: 0x06000328 RID: 808 RVA: 0x0000CE6C File Offset: 0x0000B06C
		// (set) Token: 0x06000329 RID: 809 RVA: 0x0000CE74 File Offset: 0x0000B074
		public string ColumnName { get; set; }

		// Token: 0x0600032A RID: 810 RVA: 0x0000CE7D File Offset: 0x0000B07D
		public LocalizedStringChangedEventArgs(string name, string lstr, string colName)
		{
			this.Name = name;
			this.LocalizedString = lstr;
			this.ColumnName = colName;
		}
	}
}
