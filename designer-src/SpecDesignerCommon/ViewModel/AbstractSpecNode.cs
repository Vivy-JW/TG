using System;
using System.ComponentModel;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.UndoRedoCommands;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000013 RID: 19
	public abstract class AbstractSpecNode : INotifyPropertyChanged
	{
		// Token: 0x17000014 RID: 20
		// (get) Token: 0x0600007D RID: 125 RVA: 0x00003DA9 File Offset: 0x00001FA9
		// (set) Token: 0x0600007E RID: 126 RVA: 0x00003DB1 File Offset: 0x00001FB1
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x0600007F RID: 127 RVA: 0x00003DBA File Offset: 0x00001FBA
		public AbstractSpecNode()
		{
		}

		// Token: 0x06000080 RID: 128 RVA: 0x00003DC2 File Offset: 0x00001FC2
		public AbstractSpecNode(PackageKey key, XElement source)
		{
			if (source == null)
			{
				return;
			}
			this.ProgramKey = key;
			this._source = source;
			this._status = AbstractSpecNode.GetStatusFromSource(this.GetAttribute("status"));
		}

		// Token: 0x06000081 RID: 129 RVA: 0x00003DF2 File Offset: 0x00001FF2
		public AbstractSpecNode(PackageKey key, XElement source, string src)
			: this(key, source)
		{
			this.SetAttribute("src", src);
		}

		// Token: 0x06000082 RID: 130 RVA: 0x00003E08 File Offset: 0x00002008
		internal static SpecStatus GetStatusFromSource(string status)
		{
			if (status == null)
			{
				return SpecStatus.NULL;
			}
			string text;
			if ((text = status.ToLower()) != null)
			{
				if (text == "c")
				{
					return SpecStatus.CREATE;
				}
				if (text == "u")
				{
					return SpecStatus.MODIFY;
				}
				if (text == "d")
				{
					return SpecStatus.DELETE;
				}
			}
			return SpecStatus.NULL;
		}

		// Token: 0x06000083 RID: 131 RVA: 0x00003E54 File Offset: 0x00002054
		internal void SetStatusToSource()
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
			this.SetAttribute("status", text);
		}

		// Token: 0x17000015 RID: 21
		// (get) Token: 0x06000084 RID: 132 RVA: 0x00003EAA File Offset: 0x000020AA
		// (set) Token: 0x06000085 RID: 133 RVA: 0x00003EB2 File Offset: 0x000020B2
		public XElement Source
		{
			get
			{
				return this._source;
			}
			internal set
			{
				this._source = value;
			}
		}

		// Token: 0x17000016 RID: 22
		// (get) Token: 0x06000086 RID: 134 RVA: 0x00003EBB File Offset: 0x000020BB
		// (set) Token: 0x06000087 RID: 135 RVA: 0x00003ED2 File Offset: 0x000020D2
		public bool IsCited
		{
			get
			{
				return this.GetAttribute("cite_std") != "N";
			}
			set
			{
				this.SetAttribute("cite_std", value ? "Y" : "N");
				this.OnPropertyChanged("IsCited");
			}
		}

		// Token: 0x06000088 RID: 136 RVA: 0x00003EF9 File Offset: 0x000020F9
		public void SetIsCited()
		{
			this.SetAttribute("cite_std", "Y");
		}

		// Token: 0x17000017 RID: 23
		// (get) Token: 0x06000089 RID: 137 RVA: 0x00003F0B File Offset: 0x0000210B
		// (set) Token: 0x0600008A RID: 138 RVA: 0x00003F13 File Offset: 0x00002113
		public bool IsExcluded { get; set; }

		// Token: 0x17000018 RID: 24
		// (get) Token: 0x0600008B RID: 139 RVA: 0x00003F1C File Offset: 0x0000211C
		// (set) Token: 0x0600008C RID: 140 RVA: 0x00003F2C File Offset: 0x0000212C
		public virtual string Name
		{
			get
			{
				return this.GetAttribute("name");
			}
			set
			{
				if (value == this.Name)
				{
					return;
				}
				RenameUndoRedoCommand renameUndoRedoCommand = new RenameUndoRedoCommand(this.ProgramKey, this.Name, value);
				renameUndoRedoCommand.Execute();
				if (SettingManager.Get().GetUndoRedoManager(this.ProgramKey) != null)
				{
					SettingManager.Get().GetUndoRedoManager(this.ProgramKey).AddUndo(renameUndoRedoCommand);
				}
			}
		}

		// Token: 0x0600008D RID: 141 RVA: 0x00003F89 File Offset: 0x00002189
		internal virtual void SetName(FormSpecModel model, string newName)
		{
			this.SetAttribute("name", newName);
			this.OnPropertyChanged("Name");
		}

		// Token: 0x17000019 RID: 25
		// (get) Token: 0x0600008E RID: 142 RVA: 0x00003FA2 File Offset: 0x000021A2
		// (set) Token: 0x0600008F RID: 143 RVA: 0x00003FAC File Offset: 0x000021AC
		public virtual SpecStatus Status
		{
			get
			{
				return this._status;
			}
			set
			{
				this._status = value;
				if (this.PropertyChanged != null)
				{
					this.PropertyChanged(this, new PropertyChangedEventArgs("Status"));
				}
				FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo.FindNodeByName(this.Name);
				if (formSpecModel != null && formSpecModel.GeneroComponent != null)
				{
					formSpecModel.GeneroComponent.OnPropertyChanged("SpecNodeStatus");
				}
			}
		}

		// Token: 0x1700001A RID: 26
		// (get) Token: 0x06000090 RID: 144 RVA: 0x0000401A File Offset: 0x0000221A
		public string Ver
		{
			get
			{
				return this.GetAttribute("ver");
			}
		}

		// Token: 0x1700001B RID: 27
		// (get) Token: 0x06000091 RID: 145 RVA: 0x00004027 File Offset: 0x00002227
		public string Src
		{
			get
			{
				return this.GetAttribute("src");
			}
		}

		// Token: 0x1700001C RID: 28
		// (get) Token: 0x06000092 RID: 146 RVA: 0x00004034 File Offset: 0x00002234
		public bool IsCustomized
		{
			get
			{
				return "c" == this.GetAttribute("src");
			}
		}

		// Token: 0x1700001D RID: 29
		// (get) Token: 0x06000093 RID: 147 RVA: 0x00004050 File Offset: 0x00002250
		// (set) Token: 0x06000094 RID: 148 RVA: 0x00004060 File Offset: 0x00002260
		public virtual string CDATA
		{
			get
			{
				return this.Source.Value;
			}
			set
			{
				if (this.IsCited || value == this.CDATA)
				{
					return;
				}
				SDSpecUndoRedoCommand sdspecUndoRedoCommand = new SDSpecUndoRedoCommand(this, value);
				SettingManager.Get().GetUndoRedoManager(this.ProgramKey).AddThenExecute(sdspecUndoRedoCommand);
			}
		}

		// Token: 0x1700001E RID: 30
		// (get) Token: 0x06000095 RID: 149 RVA: 0x000040A2 File Offset: 0x000022A2
		// (set) Token: 0x06000096 RID: 150 RVA: 0x000040AF File Offset: 0x000022AF
		public virtual string Widget
		{
			get
			{
				return this.GetAttribute("widget");
			}
			set
			{
				if (string.IsNullOrEmpty(value) || value == this.Widget)
				{
					return;
				}
				this.SetAttribute("widget", value);
			}
		}

		// Token: 0x1700001F RID: 31
		// (get) Token: 0x06000097 RID: 151
		public abstract XElement CitedSpec { get; }

		// Token: 0x17000020 RID: 32
		// (get) Token: 0x06000098 RID: 152 RVA: 0x00004120 File Offset: 0x00002320
		public string SASpecText
		{
			get
			{
				string specName = this.GetAttribute("column");
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo;
				XElement xelement = specificationInfo.TSDElement.Element("sa_spec");
				if (xelement == null)
				{
					return null;
				}
				XElement xelement2;
				if (this is SpecActionNode)
				{
					specName = this.GetAttribute("id");
					xelement2 = (from sa in xelement.Elements("sa_act")
						where sa.Attribute("name").Value == specName
						select sa).FirstOrDefault<XElement>();
				}
				else
				{
					xelement2 = (from sa in xelement.Elements("sa_field")
						where sa.Attribute("name").Value == specName
						select sa).FirstOrDefault<XElement>();
				}
				if (xelement2 != null)
				{
					return xelement2.Value;
				}
				return null;
			}
		}

		// Token: 0x06000099 RID: 153 RVA: 0x00004202 File Offset: 0x00002402
		public XElement ToXml()
		{
			if (SpecStatus.CREATE == this.Status)
			{
				return null;
			}
			this.SetStatusToSource();
			return this.Source;
		}

		// Token: 0x0600009A RID: 154 RVA: 0x0000421B File Offset: 0x0000241B
		public override string ToString()
		{
			this.SetStatusToSource();
			return this.Source.ToString();
		}

		// Token: 0x0600009B RID: 155 RVA: 0x00004230 File Offset: 0x00002430
		protected virtual void AttributeChanged(string key, string newValue)
		{
			if (this.IsCited)
			{
				return;
			}
			string attribute = this.GetAttribute(key);
			newValue = (string.IsNullOrEmpty(newValue) ? "" : newValue);
			if (attribute != null && attribute.Equals(newValue))
			{
				return;
			}
			this.SetAttribute(key, newValue);
			this.Status |= SpecStatus.MODIFY;
			this.OnPropertyChanged("");
		}

		// Token: 0x0600009C RID: 156 RVA: 0x0000428E File Offset: 0x0000248E
		private void PublishSpecPropertyChanged()
		{
			if (EventAggregatorManager.ContainsKey(this.ProgramKey))
			{
				EventAggregatorManager.Get(this.ProgramKey).GetEvent<SpecPropertiesChangedEvent>().Publish(this.ProgramKey);
			}
		}

		// Token: 0x14000001 RID: 1
		// (add) Token: 0x0600009D RID: 157 RVA: 0x000042B8 File Offset: 0x000024B8
		// (remove) Token: 0x0600009E RID: 158 RVA: 0x000042F0 File Offset: 0x000024F0
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x0600009F RID: 159 RVA: 0x00004325 File Offset: 0x00002525
		internal virtual void OnPropertyChanged(string propertyName)
		{
			this.Status |= SpecStatus.MODIFY;
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
			this.PublishSpecPropertyChanged();
		}

		// Token: 0x060000A0 RID: 160 RVA: 0x00004355 File Offset: 0x00002555
		public void SetAttribute(string key, string value)
		{
			if (value == null)
			{
				value = "";
			}
			this.Source.SetAttributeValue(key, value);
		}

		// Token: 0x060000A1 RID: 161 RVA: 0x00004373 File Offset: 0x00002573
		public string GetAttribute(string key)
		{
			if (this.Source != null && this.Source.Attribute(key) != null)
			{
				return this.Source.Attribute(key).Value;
			}
			return null;
		}

		// Token: 0x060000A2 RID: 162 RVA: 0x000043A8 File Offset: 0x000025A8
		public override bool Equals(object obj)
		{
			if (obj == null)
			{
				return false;
			}
			AbstractSpecNode abstractSpecNode = obj as AbstractSpecNode;
			return this.Equals(abstractSpecNode);
		}

		// Token: 0x060000A3 RID: 163 RVA: 0x000043C8 File Offset: 0x000025C8
		public bool Equals(AbstractSpecNode p)
		{
			return (p == null && this.Source == null) || (p != null && this.Source != null && this.Source.ToString() == p.Source.ToString());
		}

		// Token: 0x060000A4 RID: 164 RVA: 0x000043FF File Offset: 0x000025FF
		public override int GetHashCode()
		{
			return base.GetHashCode();
		}

		// Token: 0x04000035 RID: 53
		private XElement _source;

		// Token: 0x04000036 RID: 54
		private SpecStatus _status;
	}
}
