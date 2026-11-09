using System;
using System.ComponentModel;
using System.Windows;
using System.Xml.Linq;
using SpecDesignerCommon.Events;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000055 RID: 85
	public abstract class AbstractStringNode : INotifyPropertyChanged
	{
		// Token: 0x170000AD RID: 173
		// (get) Token: 0x060002D1 RID: 721 RVA: 0x0000C34E File Offset: 0x0000A54E
		// (set) Token: 0x060002D2 RID: 722 RVA: 0x0000C356 File Offset: 0x0000A556
		protected PackageKey ProgramKey { get; set; }

		// Token: 0x060002D3 RID: 723 RVA: 0x0000C35F File Offset: 0x0000A55F
		public AbstractStringNode()
		{
		}

		// Token: 0x060002D4 RID: 724 RVA: 0x0000C367 File Offset: 0x0000A567
		public AbstractStringNode(PackageKey key, XElement source)
		{
			this.ProgramKey = key;
			this._source = source;
			this.GetStatusFromSource();
		}

		// Token: 0x060002D5 RID: 725 RVA: 0x0000C384 File Offset: 0x0000A584
		private void GetStatusFromSource()
		{
			string text;
			if (this.Source.Attribute("lstr") != null && (text = this.Source.Attribute("lstr").Value.ToLower()) != null)
			{
				if (text == "u")
				{
					this.Status = SpecStatus.MODIFY;
					return;
				}
				if (!(text == "d"))
				{
					return;
				}
				this.Status = SpecStatus.DELETE;
			}
		}

		// Token: 0x060002D6 RID: 726 RVA: 0x0000C3F8 File Offset: 0x0000A5F8
		private void SetStatusToSource()
		{
			string text = "";
			if ((this.Status & SpecStatus.DELETE) == SpecStatus.DELETE)
			{
				text = "d";
			}
			else if ((this.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
			{
				text = "u";
			}
			else if ((this.Status & SpecStatus.CREATE) == SpecStatus.CREATE)
			{
				text = "c";
			}
			this.SetAttribute("lstr", text);
		}

		// Token: 0x170000AE RID: 174
		// (get) Token: 0x060002D7 RID: 727 RVA: 0x0000C44E File Offset: 0x0000A64E
		public XElement Source
		{
			get
			{
				return this._source;
			}
		}

		// Token: 0x170000AF RID: 175
		// (get) Token: 0x060002D8 RID: 728 RVA: 0x0000C456 File Offset: 0x0000A656
		// (set) Token: 0x060002D9 RID: 729 RVA: 0x0000C464 File Offset: 0x0000A664
		public string Name
		{
			get
			{
				return this.GetAttribute("name");
			}
			set
			{
				string name = this.Name;
				if (name == value)
				{
					return;
				}
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo;
				if (specificationInfo.FormSpeDictionary.ContainsKey(value))
				{
					throw new Exception(Application.Current.FindResource("Message_NameAlreadyExist") as string);
				}
				this.AttributeChanged("name", value);
				specificationInfo.Rename(name, value);
			}
		}

		// Token: 0x170000B0 RID: 176
		// (get) Token: 0x060002DA RID: 730 RVA: 0x0000C4D4 File Offset: 0x0000A6D4
		// (set) Token: 0x060002DB RID: 731 RVA: 0x0000C4E1 File Offset: 0x0000A6E1
		public string Text
		{
			get
			{
				return this.GetAttribute("text");
			}
			set
			{
				this.AttributeChanged("text", value);
			}
		}

		// Token: 0x170000B1 RID: 177
		// (get) Token: 0x060002DC RID: 732 RVA: 0x0000C4EF File Offset: 0x0000A6EF
		// (set) Token: 0x060002DD RID: 733 RVA: 0x0000C4F7 File Offset: 0x0000A6F7
		public SpecStatus Status
		{
			get
			{
				return this._status;
			}
			set
			{
				this._status = value;
				this.OnPropertyChanged("Status");
			}
		}

		// Token: 0x060002DE RID: 734 RVA: 0x0000C50B File Offset: 0x0000A70B
		private void PublishSpecPropertyChanged()
		{
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<SpecPropertiesChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x060002DF RID: 735 RVA: 0x0000C528 File Offset: 0x0000A728
		internal void AttributeChanged(string key, string newValue)
		{
			string attribute = this.GetAttribute(key);
			newValue = (string.IsNullOrEmpty(newValue) ? "" : newValue);
			if (attribute != null && attribute.Equals(newValue))
			{
				return;
			}
			this.SetAttribute(key, newValue);
			this.Status |= SpecStatus.MODIFY;
			this.SpecPropertyChanged(key, attribute, newValue);
		}

		// Token: 0x060002E0 RID: 736 RVA: 0x0000C57B File Offset: 0x0000A77B
		public XElement ToXml()
		{
			if (SpecStatus.CREATE == this.Status)
			{
				return null;
			}
			this.SetStatusToSource();
			return this.Source;
		}

		// Token: 0x060002E1 RID: 737 RVA: 0x0000C594 File Offset: 0x0000A794
		public override string ToString()
		{
			XElement xelement = this.ToXml();
			if (xelement != null)
			{
				return xelement.ToString();
			}
			return null;
		}

		// Token: 0x060002E2 RID: 738 RVA: 0x0000C5B3 File Offset: 0x0000A7B3
		public virtual void SpecPropertyChanged(string propertyName, string oldValue, string newValue)
		{
			this.OnPropertyChanged("");
			this.PublishSpecPropertyChanged();
		}

		// Token: 0x1400000A RID: 10
		// (add) Token: 0x060002E3 RID: 739 RVA: 0x0000C5C8 File Offset: 0x0000A7C8
		// (remove) Token: 0x060002E4 RID: 740 RVA: 0x0000C600 File Offset: 0x0000A800
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060002E5 RID: 741 RVA: 0x0000C635 File Offset: 0x0000A835
		public void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x060002E6 RID: 742 RVA: 0x0000C651 File Offset: 0x0000A851
		public void SetAttribute(string key, string value)
		{
			if (value == null)
			{
				value = "";
			}
			this.Source.Attribute(key).Value = value;
		}

		// Token: 0x060002E7 RID: 743 RVA: 0x0000C674 File Offset: 0x0000A874
		public string GetAttribute(string key)
		{
			if (this.Source != null && this.Source.Attribute(key) != null)
			{
				return this.Source.Attribute(key).Value;
			}
			return null;
		}

		// Token: 0x060002E8 RID: 744 RVA: 0x0000C6AC File Offset: 0x0000A8AC
		public override bool Equals(object obj)
		{
			if (obj == null)
			{
				return false;
			}
			AbstractStringNode abstractStringNode = obj as AbstractStringNode;
			return this.Equals(abstractStringNode);
		}

		// Token: 0x060002E9 RID: 745 RVA: 0x0000C6CC File Offset: 0x0000A8CC
		public bool Equals(AbstractStringNode p)
		{
			return p != null && this.Source.ToString() == p.Source.ToString();
		}

		// Token: 0x060002EA RID: 746 RVA: 0x0000C6EE File Offset: 0x0000A8EE
		public override int GetHashCode()
		{
			return base.GetHashCode();
		}

		// Token: 0x0400010E RID: 270
		private XElement _source;

		// Token: 0x0400010F RID: 271
		private SpecStatus _status;
	}
}
