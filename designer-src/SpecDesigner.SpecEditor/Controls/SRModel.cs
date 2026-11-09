using System;
using System.Xml.Linq;
using SpecDesignerCommon;

namespace SpecDesigner.SpecEditor.Controls
{
	// Token: 0x0200001E RID: 30
	public class SRModel : BaseModel
	{
		// Token: 0x060000F3 RID: 243 RVA: 0x00009384 File Offset: 0x00007584
		public SRModel(XElement xml, PackageKey programKey)
			: base(programKey)
		{
			this._xml = xml;
			this.SRName = xml.Attribute("name").Value;
		}

		// Token: 0x1700001D RID: 29
		// (get) Token: 0x060000F4 RID: 244 RVA: 0x000093AF File Offset: 0x000075AF
		// (set) Token: 0x060000F5 RID: 245 RVA: 0x000093B7 File Offset: 0x000075B7
		public string SRName { get; private set; }

		// Token: 0x1700001E RID: 30
		// (get) Token: 0x060000F6 RID: 246 RVA: 0x000093C0 File Offset: 0x000075C0
		// (set) Token: 0x060000F7 RID: 247 RVA: 0x000093F9 File Offset: 0x000075F9
		public string Cascade
		{
			get
			{
				if (this._xml.Attribute("cascade") != null)
				{
					return this._xml.Attribute("cascade").Value;
				}
				return "";
			}
			set
			{
				this._xml.SetAttributeValue("cascade", value);
				base.NotifyPropertyChanged("Cascade");
			}
		}

		// Token: 0x0400008E RID: 142
		private XElement _xml;
	}
}
