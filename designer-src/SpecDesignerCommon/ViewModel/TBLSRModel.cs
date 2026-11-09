using System;
using System.ComponentModel;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000053 RID: 83
	public class TBLSRModel : IDisposable, INotifyPropertyChanged
	{
		// Token: 0x170000A6 RID: 166
		// (get) Token: 0x060002BD RID: 701 RVA: 0x0000BFED File Offset: 0x0000A1ED
		// (set) Token: 0x060002BE RID: 702 RVA: 0x0000BFF5 File Offset: 0x0000A1F5
		public PackageKey Key { get; private set; }

		// Token: 0x060002BF RID: 703 RVA: 0x0000BFFE File Offset: 0x0000A1FE
		public TBLSRModel(XElement xml, PackageKey key)
		{
			this.xml = xml;
			this.Key = key;
			this.xml.Changed += this.xml_Changed;
		}

		// Token: 0x170000A7 RID: 167
		// (get) Token: 0x060002C0 RID: 704 RVA: 0x0000C02B File Offset: 0x0000A22B
		// (set) Token: 0x060002C1 RID: 705 RVA: 0x0000C064 File Offset: 0x0000A264
		public string Cascade
		{
			get
			{
				if (this.xml.Attribute("cascade") == null)
				{
					return "N";
				}
				return this.xml.Attribute("cascade").Value;
			}
			set
			{
				this.xml.SetAttributeValue("cascade", value);
				this.NotifyPropertyChanged("Cascade");
			}
		}

		// Token: 0x170000A8 RID: 168
		// (get) Token: 0x060002C2 RID: 706 RVA: 0x0000C087 File Offset: 0x0000A287
		public string programName
		{
			get
			{
				return this.Key.Program;
			}
		}

		// Token: 0x170000A9 RID: 169
		// (get) Token: 0x060002C3 RID: 707 RVA: 0x0000C094 File Offset: 0x0000A294
		// (set) Token: 0x060002C4 RID: 708 RVA: 0x0000C0CD File Offset: 0x0000A2CD
		public string Kind
		{
			get
			{
				if (this.xml.Attribute("kind") == null)
				{
					return "Table";
				}
				return this.xml.Attribute("kind").Value;
			}
			set
			{
			}
		}

		// Token: 0x170000AA RID: 170
		// (get) Token: 0x060002C5 RID: 709 RVA: 0x0000C0CF File Offset: 0x0000A2CF
		public string SRName
		{
			get
			{
				return this.xml.Attribute("name").Value;
			}
		}

		// Token: 0x170000AB RID: 171
		// (get) Token: 0x060002C6 RID: 710 RVA: 0x0000C0EB File Offset: 0x0000A2EB
		// (set) Token: 0x060002C7 RID: 711 RVA: 0x0000C0F3 File Offset: 0x0000A2F3
		public TBLModel TBL { get; set; }

		// Token: 0x060002C8 RID: 712 RVA: 0x0000C0FC File Offset: 0x0000A2FC
		public void Dispose()
		{
			this.xml.Changed -= this.xml_Changed;
			this.xml = null;
		}

		// Token: 0x060002C9 RID: 713 RVA: 0x0000C11C File Offset: 0x0000A31C
		public void SetAttribute(string attr, string value)
		{
			this.xml.SetAttributeValue(attr, value);
		}

		// Token: 0x060002CA RID: 714 RVA: 0x0000C130 File Offset: 0x0000A330
		private void xml_Changed(object sender, XObjectChangeEventArgs e)
		{
			if (sender is XAttribute && e.ObjectChange == XObjectChange.Value)
			{
				if ((sender as XAttribute).Name == "status" && (sender as XAttribute).Value == ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE))
				{
					this.TBL.renderSRs();
				}
				if ((sender as XAttribute).Name == "kind")
				{
					this.NotifyPropertyChanged("Kind");
				}
			}
		}

		// Token: 0x14000009 RID: 9
		// (add) Token: 0x060002CB RID: 715 RVA: 0x0000C1BC File Offset: 0x0000A3BC
		// (remove) Token: 0x060002CC RID: 716 RVA: 0x0000C1F4 File Offset: 0x0000A3F4
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060002CD RID: 717 RVA: 0x0000C229 File Offset: 0x0000A429
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x0400010A RID: 266
		private XElement xml;
	}
}
