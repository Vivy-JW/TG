using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Windows;
using System.Xml.Linq;
using SpecDesignerCommon.UndoRedoCommands;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000A4 RID: 164
	public class ProgRelProgram : INotifyPropertyChanged
	{
		// Token: 0x170001F6 RID: 502
		// (get) Token: 0x060006E7 RID: 1767 RVA: 0x0001F11E File Offset: 0x0001D31E
		// (set) Token: 0x060006E8 RID: 1768 RVA: 0x0001F126 File Offset: 0x0001D326
		public XElement Source { get; private set; }

		// Token: 0x060006E9 RID: 1769 RVA: 0x0001F12F File Offset: 0x0001D32F
		public ProgRelProgram(SpecProgRelNode progrel, XElement source)
		{
			this.Source = source;
			this._progrel = progrel;
		}

		// Token: 0x060006EA RID: 1770 RVA: 0x0001F148 File Offset: 0x0001D348
		public static ProgRelProgram Create(SpecProgRelNode parent)
		{
			XElement xelement = new XElement("program", new object[]
			{
				new XAttribute("name", ""),
				new XAttribute("type", ""),
				new XAttribute("order", parent.getMaxOrder() + 1)
			});
			return new ProgRelProgram(parent, xelement);
		}

		// Token: 0x060006EB RID: 1771 RVA: 0x0001F1C1 File Offset: 0x0001D3C1
		public void SetAttribute(string key, string value)
		{
			if (value == null)
			{
				value = "";
			}
			this.Source.SetAttributeValue(key, value);
		}

		// Token: 0x060006EC RID: 1772 RVA: 0x0001F1DF File Offset: 0x0001D3DF
		public XElement ToXml()
		{
			return this.Source;
		}

		// Token: 0x170001F7 RID: 503
		// (get) Token: 0x060006ED RID: 1773 RVA: 0x0001F1E7 File Offset: 0x0001D3E7
		// (set) Token: 0x060006EE RID: 1774 RVA: 0x0001F203 File Offset: 0x0001D403
		public string Program
		{
			get
			{
				return this.Source.Attribute("name").Value;
			}
			set
			{
				this.AttributeChanged("name", value);
			}
		}

		// Token: 0x170001F8 RID: 504
		// (get) Token: 0x060006EF RID: 1775 RVA: 0x0001F211 File Offset: 0x0001D411
		// (set) Token: 0x060006F0 RID: 1776 RVA: 0x0001F22D File Offset: 0x0001D42D
		public string Type
		{
			get
			{
				return this.Source.Attribute("type").Value;
			}
			set
			{
				this.AttributeChanged("type", value);
			}
		}

		// Token: 0x170001F9 RID: 505
		// (get) Token: 0x060006F1 RID: 1777 RVA: 0x0001F23B File Offset: 0x0001D43B
		// (set) Token: 0x060006F2 RID: 1778 RVA: 0x0001F257 File Offset: 0x0001D457
		public string Order
		{
			get
			{
				return this.Source.Attribute("order").Value;
			}
			set
			{
				this.AttributeChanged("order", value);
			}
		}

		// Token: 0x170001FA RID: 506
		// (get) Token: 0x060006F3 RID: 1779 RVA: 0x0001F268 File Offset: 0x0001D468
		public List<ProgRelType> Types
		{
			get
			{
				if (this._types == null)
				{
					this._types = new List<ProgRelType>
					{
						new ProgRelType
						{
							Type = "1",
							Text = (Application.Current.FindResource("specProperty_prog_rel_type1") as string)
						},
						new ProgRelType
						{
							Type = "2",
							Text = (Application.Current.FindResource("specProperty_prog_rel_type2") as string)
						}
					};
				}
				return this._types;
			}
		}

		// Token: 0x170001FB RID: 507
		// (get) Token: 0x060006F4 RID: 1780 RVA: 0x0001F308 File Offset: 0x0001D508
		public string TypeName
		{
			get
			{
				ProgRelType progRelType = this.Types.Where<ProgRelType>((ProgRelType t) => t.Type == this.Type).FirstOrDefault<ProgRelType>();
				if (progRelType != null)
				{
					return progRelType.Text;
				}
				return "";
			}
		}

		// Token: 0x060006F5 RID: 1781 RVA: 0x0001F344 File Offset: 0x0001D544
		private void AttributeChanged(string key, string value)
		{
			if (this.Source.Attribute(key) == null)
			{
				return;
			}
			ProgRelProgramAttributeUndoRedoCommand progRelProgramAttributeUndoRedoCommand = new ProgRelProgramAttributeUndoRedoCommand(this, this._progrel);
			progRelProgramAttributeUndoRedoCommand.AddAttributeChanged(key, this.Source.Attribute(key).Value, value);
			progRelProgramAttributeUndoRedoCommand.Execute();
			if (SettingManager.Get().GetUndoRedoManager(this._progrel.ProgramKey) != null)
			{
				SettingManager.Get().GetUndoRedoManager(this._progrel.ProgramKey).AddUndo(progRelProgramAttributeUndoRedoCommand);
			}
		}

		// Token: 0x14000019 RID: 25
		// (add) Token: 0x060006F6 RID: 1782 RVA: 0x0001F3C8 File Offset: 0x0001D5C8
		// (remove) Token: 0x060006F7 RID: 1783 RVA: 0x0001F400 File Offset: 0x0001D600
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060006F8 RID: 1784 RVA: 0x0001F435 File Offset: 0x0001D635
		public void OnPropertyChanged(string property)
		{
			this._progrel.Status = SpecStatus.MODIFY;
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x04000280 RID: 640
		private SpecProgRelNode _progrel;

		// Token: 0x04000281 RID: 641
		private List<ProgRelType> _types;
	}
}
