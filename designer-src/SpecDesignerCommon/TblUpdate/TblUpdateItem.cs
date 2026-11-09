using System;
using System.IO;
using System.Xml.Linq;

namespace SpecDesignerCommon.TblUpdate
{
	// Token: 0x02000140 RID: 320
	public class TblUpdateItem : IComparable<TblUpdateItem>
	{
		// Token: 0x170002ED RID: 749
		// (get) Token: 0x06000B2A RID: 2858 RVA: 0x000360DC File Offset: 0x000342DC
		// (set) Token: 0x06000B2B RID: 2859 RVA: 0x000360E4 File Offset: 0x000342E4
		public string Name { get; set; }

		// Token: 0x170002EE RID: 750
		// (get) Token: 0x06000B2C RID: 2860 RVA: 0x000360ED File Offset: 0x000342ED
		// (set) Token: 0x06000B2D RID: 2861 RVA: 0x000360F5 File Offset: 0x000342F5
		public string Module { get; set; }

		// Token: 0x170002EF RID: 751
		// (get) Token: 0x06000B2E RID: 2862 RVA: 0x000360FE File Offset: 0x000342FE
		// (set) Token: 0x06000B2F RID: 2863 RVA: 0x00036106 File Offset: 0x00034306
		public string Lang { get; set; }

		// Token: 0x170002F0 RID: 752
		// (get) Token: 0x06000B30 RID: 2864 RVA: 0x0003610F File Offset: 0x0003430F
		public string LocalPath
		{
			get
			{
				return Path.Combine(this.Module.ToLower(), "tbl", this.Name);
			}
		}

		// Token: 0x170002F1 RID: 753
		// (get) Token: 0x06000B31 RID: 2865 RVA: 0x0003612C File Offset: 0x0003432C
		public string RemotePath
		{
			get
			{
				return Path.Combine("tbl", this.Lang, this.Name);
			}
		}

		// Token: 0x06000B32 RID: 2866 RVA: 0x00036144 File Offset: 0x00034344
		public TblUpdateItem(XElement source)
		{
			this.Name = ((source.Attribute("name") != null) ? source.Attribute("name").Value : string.Empty);
			this.Module = ((source.Attribute("module") != null) ? source.Attribute("module").Value : string.Empty);
			this.Lang = ((source.Attribute("lang") != null) ? source.Attribute("lang").Value : string.Empty);
		}

		// Token: 0x06000B33 RID: 2867 RVA: 0x000361F3 File Offset: 0x000343F3
		public TblUpdateItem(string name, string module, string lang)
		{
			this.Name = name;
			this.Module = module;
			this.Lang = lang;
		}

		// Token: 0x06000B34 RID: 2868 RVA: 0x00036210 File Offset: 0x00034410
		public int CompareTo(TblUpdateItem other)
		{
			if (other.Name == this.Name && other.Module == this.Module && other.Lang == this.Lang)
			{
				return 0;
			}
			return 1;
		}
	}
}
