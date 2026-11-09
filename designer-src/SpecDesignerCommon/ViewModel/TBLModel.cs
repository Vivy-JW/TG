using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Linq;
using System.Windows;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000027 RID: 39
	public class TBLModel : INotifyPropertyChanged, IDisposable
	{
		// Token: 0x17000042 RID: 66
		// (get) Token: 0x0600012E RID: 302 RVA: 0x00006454 File Offset: 0x00004654
		// (set) Token: 0x0600012F RID: 303 RVA: 0x0000645C File Offset: 0x0000465C
		public PackageKey Key { get; private set; }

		// Token: 0x06000130 RID: 304 RVA: 0x00006468 File Offset: 0x00004668
		public TBLModel(XElement xml, TableAssociationModel tableAssociationModel)
		{
			this.tableAssociationModel = tableAssociationModel;
			this.xml = xml;
			this.Key = tableAssociationModel.Key;
			if (this.xml.Attribute("pk") == null)
			{
				this.xml.SetAttributeValue("pk", "");
			}
			if (this.xml.Attribute("fk_master") == null)
			{
				this.xml.SetAttributeValue("fk_master", "");
			}
			if (this.xml.Attribute("fk_detail") == null)
			{
				this.xml.SetAttributeValue("fk_detail", "");
			}
			this.renderSRs();
			this.xml.Changed += this.xml_Changed;
		}

		// Token: 0x14000005 RID: 5
		// (add) Token: 0x06000131 RID: 305 RVA: 0x00006558 File Offset: 0x00004758
		// (remove) Token: 0x06000132 RID: 306 RVA: 0x00006590 File Offset: 0x00004790
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x17000043 RID: 67
		// (get) Token: 0x06000133 RID: 307 RVA: 0x000065C5 File Offset: 0x000047C5
		// (set) Token: 0x06000134 RID: 308 RVA: 0x000065E4 File Offset: 0x000047E4
		public string FkDetail
		{
			get
			{
				return this.xml.Attribute("fk_detail").Value;
			}
			set
			{
				string text = ((this.xml.Attribute("fk_detail") != null) ? this.xml.Attribute("fk_detail").Value : null);
				if (text == value)
				{
					return;
				}
				this.xml.SetAttributeValue("fk_detail", value);
				this.NotifyPropertyChanged("FkDetail", text, value);
			}
		}

		// Token: 0x17000044 RID: 68
		// (get) Token: 0x06000135 RID: 309 RVA: 0x00006655 File Offset: 0x00004855
		// (set) Token: 0x06000136 RID: 310 RVA: 0x00006674 File Offset: 0x00004874
		public string FkMaster
		{
			get
			{
				return this.xml.Attribute("fk_master").Value;
			}
			set
			{
				string text = ((this.xml.Attribute("fk_master") != null) ? this.xml.Attribute("fk_master").Value : null);
				if (text == value)
				{
					return;
				}
				this.xml.SetAttributeValue("fk_master", value);
				this.NotifyPropertyChanged("FkMaster", text, value);
			}
		}

		// Token: 0x17000045 RID: 69
		// (get) Token: 0x06000137 RID: 311 RVA: 0x000066E8 File Offset: 0x000048E8
		public int Level
		{
			get
			{
				if (this._level == -1)
				{
					this._level = 0;
					for (XElement xelement = this.getParentByParentName(this.Parent); xelement != null; xelement = this.getParentByParentName(xelement.Attribute("parent").Value))
					{
						this._level++;
					}
				}
				return this._level;
			}
		}

		// Token: 0x17000046 RID: 70
		// (get) Token: 0x06000138 RID: 312 RVA: 0x00006747 File Offset: 0x00004947
		public string Main
		{
			get
			{
				if (this.xml.Attribute("main") == null)
				{
					return "N";
				}
				return this.xml.Attribute("main").Value;
			}
		}

		// Token: 0x17000047 RID: 71
		// (get) Token: 0x06000139 RID: 313 RVA: 0x00006780 File Offset: 0x00004980
		// (set) Token: 0x0600013A RID: 314 RVA: 0x0000679C File Offset: 0x0000499C
		public string Parent
		{
			get
			{
				return this.xml.Attribute("parent").Value;
			}
			set
			{
				string text = ((this.xml.Attribute("parent") != null) ? this.xml.Attribute("parent").Value : "");
				this.xml.SetAttributeValue("parent", value);
				this.NotifyPropertyChanged("Parent", text, value);
			}
		}

		// Token: 0x17000048 RID: 72
		// (get) Token: 0x0600013B RID: 315 RVA: 0x00006807 File Offset: 0x00004A07
		// (set) Token: 0x0600013C RID: 316 RVA: 0x00006824 File Offset: 0x00004A24
		public string PK
		{
			get
			{
				return this.xml.Attribute("pk").Value;
			}
			set
			{
				string text = ((this.xml.Attribute("pk") != null) ? this.xml.Attribute("pk").Value : "");
				if (text == value)
				{
					return;
				}
				this.xml.SetAttributeValue("pk", value);
				this.NotifyPropertyChanged("PK", text, value);
			}
		}

		// Token: 0x17000049 RID: 73
		// (get) Token: 0x0600013D RID: 317 RVA: 0x00006899 File Offset: 0x00004A99
		public string programName
		{
			get
			{
				return this.Key.Program;
			}
		}

		// Token: 0x1700004A RID: 74
		// (get) Token: 0x0600013E RID: 318 RVA: 0x000068A6 File Offset: 0x00004AA6
		public ObservableCollection<TBLSRModel> ScreenRecords
		{
			get
			{
				return this._screenRecords;
			}
		}

		// Token: 0x1700004B RID: 75
		// (get) Token: 0x0600013F RID: 319 RVA: 0x000068AE File Offset: 0x00004AAE
		// (set) Token: 0x06000140 RID: 320 RVA: 0x000068CC File Offset: 0x00004ACC
		public string Status
		{
			get
			{
				return this.xml.Attribute("status").Value;
			}
			set
			{
				if (this.xml.Attribute("status") != null)
				{
					string value2 = this.xml.Attribute("status").Value;
				}
				this.xml.SetAttributeValue("status", value);
				this.NotifyPropertyChanged("Status", null, null);
			}
		}

		// Token: 0x1700004C RID: 76
		// (get) Token: 0x06000141 RID: 321 RVA: 0x00006930 File Offset: 0x00004B30
		// (set) Token: 0x06000142 RID: 322 RVA: 0x000069AC File Offset: 0x00004BAC
		public string TBLName
		{
			get
			{
				return this.xml.Attribute("name").Value;
			}
			set
			{
				string oldName = this.xml.Attribute("name").Value;
				if (oldName.Trim().Equals(value.Trim()))
				{
					return;
				}
				if (this.xml.Attribute("main") != null && this.xml.Attribute("main").Value == "Y")
				{
					DesignerMessageBox.Show(Application.Current.FindResource("Message_MainTableCantChangeName") as string);
					return;
				}
				if (oldName != "")
				{
					XElement xelement = new XElement(this.xml);
					xelement.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE));
					foreach (XElement xelement2 in xelement.Elements())
					{
						xelement2.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE));
					}
					this.xml.Parent.Add(xelement);
					IEnumerable<XElement> enumerable = from t in this.xml.Parent.Elements("tbl")
						where t.Attribute("parent").Value == oldName && t.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
						select t;
					foreach (XElement xelement3 in enumerable)
					{
						xelement3.SetAttributeValue("parent", value);
						xelement3.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY));
					}
					foreach (XElement xelement4 in this.xml.Elements("sr"))
					{
						xelement4.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY));
					}
				}
				this.xml.SetAttributeValue("name", value);
				this.NotifyPropertyChanged("TBLName", oldName, value);
				this.NotifyPropertyChanged("TBLText", null, null);
				this.NotifyPropertyChanged("TBLDesc", null, null);
			}
		}

		// Token: 0x1700004D RID: 77
		// (get) Token: 0x06000143 RID: 323 RVA: 0x00006C3C File Offset: 0x00004E3C
		public string TBLDesc
		{
			get
			{
				if (!string.IsNullOrEmpty(this.TBLName))
				{
					return TableColumnHelper.GetTableDesc(this.TBLName);
				}
				return string.Empty;
			}
		}

		// Token: 0x1700004E RID: 78
		// (get) Token: 0x06000144 RID: 324 RVA: 0x00006C5C File Offset: 0x00004E5C
		// (set) Token: 0x06000145 RID: 325 RVA: 0x00006C98 File Offset: 0x00004E98
		public string UpperTable
		{
			get
			{
				if (this.xml.Attribute("upper_table") == null)
				{
					return string.Empty;
				}
				return this.xml.Attribute("upper_table").Value;
			}
			set
			{
				string upperTable = this.UpperTable;
				if (upperTable == value)
				{
					return;
				}
				this.xml.SetAttributeValue("upper_table", value);
				this.NotifyPropertyChanged("UpperTable", upperTable, value);
			}
		}

		// Token: 0x1700004F RID: 79
		// (get) Token: 0x06000146 RID: 326 RVA: 0x00006CD9 File Offset: 0x00004ED9
		// (set) Token: 0x06000147 RID: 327 RVA: 0x00006D14 File Offset: 0x00004F14
		public string ThisKey
		{
			get
			{
				if (this.xml.Attribute("this_key") == null)
				{
					return string.Empty;
				}
				return this.xml.Attribute("this_key").Value;
			}
			set
			{
				string thisKey = this.ThisKey;
				if (thisKey == value)
				{
					return;
				}
				this.xml.SetAttributeValue("this_key", value);
				this.NotifyPropertyChanged("ThisKey", thisKey, value);
			}
		}

		// Token: 0x17000050 RID: 80
		// (get) Token: 0x06000148 RID: 328 RVA: 0x00006D55 File Offset: 0x00004F55
		// (set) Token: 0x06000149 RID: 329 RVA: 0x00006D90 File Offset: 0x00004F90
		public string UpperKey
		{
			get
			{
				if (this.xml.Attribute("upper_key") == null)
				{
					return string.Empty;
				}
				return this.xml.Attribute("upper_key").Value;
			}
			set
			{
				string upperKey = this.UpperKey;
				if (upperKey == value)
				{
					return;
				}
				this.xml.SetAttributeValue("upper_key", value);
				this.NotifyPropertyChanged("UpperKey", upperKey, value);
			}
		}

		// Token: 0x17000051 RID: 81
		// (get) Token: 0x0600014A RID: 330 RVA: 0x00006DD1 File Offset: 0x00004FD1
		public string TBLText
		{
			get
			{
				return TableColumnHelper.GetTableDesc(this.TBLName);
			}
		}

		// Token: 0x17000052 RID: 82
		// (get) Token: 0x0600014B RID: 331 RVA: 0x00006DDE File Offset: 0x00004FDE
		public XElement XML
		{
			get
			{
				return this.xml;
			}
		}

		// Token: 0x0600014C RID: 332 RVA: 0x00006DE6 File Offset: 0x00004FE6
		public void DeleteFromPersistence()
		{
			this.xml.Remove();
			this.tableAssociationModel.tbls.Remove(this);
		}

		// Token: 0x0600014D RID: 333 RVA: 0x00006E08 File Offset: 0x00005008
		public void Dispose()
		{
			this.xml.Changed -= this.xml_Changed;
			foreach (TBLSRModel tblsrmodel in this._screenRecords)
			{
				tblsrmodel.Dispose();
			}
			this._screenRecords.Clear();
			this.tableAssociationModel = null;
			this.xml = null;
		}

		// Token: 0x0600014E RID: 334 RVA: 0x00006E84 File Offset: 0x00005084
		public void NotifyPropertyChanged(string property, string oldValue, string newValue)
		{
			if (property == null || !(property == "Status"))
			{
				this.xml.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY));
			}
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
			EventAggregatorManager.Get(this.Key).GetEvent<SpecPropertiesChangedEvent>().Publish(this.Key);
		}

		// Token: 0x0600014F RID: 335 RVA: 0x00006EF8 File Offset: 0x000050F8
		public void SetAttribute(string attr, string value)
		{
			this.xml.SetAttributeValue(attr, value);
		}

		// Token: 0x06000150 RID: 336 RVA: 0x00006F0C File Offset: 0x0000510C
		internal void renderSRs()
		{
			foreach (TBLSRModel tblsrmodel in this._screenRecords)
			{
				tblsrmodel.Dispose();
			}
			this._screenRecords.Clear();
			foreach (XElement xelement in this.xml.Elements("sr"))
			{
				if (xelement.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE))
				{
					this._screenRecords.Add(new TBLSRModel(xelement, this.Key)
					{
						TBL = this
					});
				}
			}
		}

		// Token: 0x06000151 RID: 337 RVA: 0x00007020 File Offset: 0x00005220
		private XElement getParentByParentName(string p)
		{
			return (from x in this.xml.Parent.Elements("tbl")
				where x.Attribute("name").Value == p
				select x).FirstOrDefault<XElement>();
		}

		// Token: 0x06000152 RID: 338 RVA: 0x0000706A File Offset: 0x0000526A
		private void xml_Changed(object sender, XObjectChangeEventArgs e)
		{
			if (e.ObjectChange == XObjectChange.Add || e.ObjectChange == XObjectChange.Remove)
			{
				this.renderSRs();
			}
		}

		// Token: 0x06000153 RID: 339 RVA: 0x00007084 File Offset: 0x00005284
		internal bool Containas(string name)
		{
			foreach (TBLSRModel tblsrmodel in this._screenRecords)
			{
				if (tblsrmodel.SRName == name)
				{
					return true;
				}
			}
			return false;
		}

		// Token: 0x06000154 RID: 340 RVA: 0x000070E0 File Offset: 0x000052E0
		internal void Remove(string recoredName)
		{
			for (int i = 0; i < this._screenRecords.Count; i++)
			{
				TBLSRModel tblsrmodel = this._screenRecords[i];
				if (tblsrmodel.SRName == recoredName)
				{
					tblsrmodel.SetAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE));
				}
			}
		}

		// Token: 0x0400007E RID: 126
		private int _level = -1;

		// Token: 0x0400007F RID: 127
		private ObservableCollection<TBLSRModel> _screenRecords = new ObservableCollection<TBLSRModel>();

		// Token: 0x04000080 RID: 128
		private TableAssociationModel tableAssociationModel;

		// Token: 0x04000081 RID: 129
		private XElement xml;
	}
}
