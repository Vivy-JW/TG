using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000124 RID: 292
	public class FormAttributesUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x06000A3F RID: 2623 RVA: 0x00033030 File Offset: 0x00031230
		public FormAttributesUndoRedoCommand(XmlElement element, string attribute, string newValue)
		{
			this._element = element;
			this._list = new List<FormAttributesUndoRedoCommand.Property>();
			this._list.Add(new FormAttributesUndoRedoCommand.Property(attribute, element.GetAttribute(attribute), newValue));
			if (attribute != null)
			{
				if (!(attribute == "width"))
				{
					if (!(attribute == "height"))
					{
						return;
					}
					if (XmlElement.NOSET == newValue)
					{
						this._list.Add(new FormAttributesUndoRedoCommand.Property("unitHeight", element.GetAttribute("unitHeight"), XmlElement.NOSET));
					}
				}
				else if (XmlElement.NOSET == newValue)
				{
					this._list.Add(new FormAttributesUndoRedoCommand.Property("unitWidth", element.GetAttribute("unitWidth"), XmlElement.NOSET));
					return;
				}
			}
		}

		// Token: 0x06000A40 RID: 2624 RVA: 0x000330F8 File Offset: 0x000312F8
		public override void Execute()
		{
			if (this._list.Count == 0)
			{
				return;
			}
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(this._element.Key).StartGroup(formSizeComplexUndoRedoCommand);
			foreach (FormAttributesUndoRedoCommand.Property property in this._list)
			{
				string attribute;
				if ((attribute = property.Attribute) != null)
				{
					if (attribute == "rowCount" || attribute == "rowHeight" || attribute == "totalRows")
					{
						this._element.SetAttribute(property.Attribute, property.NewValue);
						this.OnPropertyChanged(property.Attribute);
						this._element.MeasureSize();
						continue;
					}
					if (attribute == "gridWidth")
					{
						this._element.GridWidth = int.Parse(property.NewValue);
						continue;
					}
					if (attribute == "gridHeight")
					{
						this._element.GridHeight = int.Parse(property.NewValue);
						continue;
					}
				}
				this._element.SetAttribute(property.Attribute, property.NewValue);
				this.OnPropertyChanged(property.Attribute);
			}
			SettingManager.Get().GetUndoRedoManager(this._element.Key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._element.Key).AddSelection(this._element, false);
		}

		// Token: 0x06000A41 RID: 2625 RVA: 0x00033284 File Offset: 0x00031484
		public override void Undo()
		{
			if (this._list.Count == 0)
			{
				return;
			}
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = null;
			foreach (FormAttributesUndoRedoCommand.Property property in this._list)
			{
				string attribute;
				if ((attribute = property.Attribute) != null && attribute == "totalRows")
				{
					formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
					SettingManager.Get().GetUndoRedoManager(this._element.Key).StartGroup(formSizeComplexUndoRedoCommand);
					this._element.SetAttribute(property.Attribute, property.OldValue);
				}
				else
				{
					this._element.SetAttribute(property.Attribute, property.OldValue);
				}
				this.OnPropertyChanged(property.Attribute);
			}
			if (formSizeComplexUndoRedoCommand != null)
			{
				SettingManager.Get().GetUndoRedoManager(this._element.Key).EndGroup(formSizeComplexUndoRedoCommand);
			}
			ComponentHelper.Get(this._element.Key).AddSelection(this._element, false);
		}

		// Token: 0x06000A42 RID: 2626 RVA: 0x00033394 File Offset: 0x00031594
		public override void Clear()
		{
			if (this._list != null)
			{
				this._list.Clear();
			}
		}

		// Token: 0x06000A43 RID: 2627 RVA: 0x000333AC File Offset: 0x000315AC
		private void OnPropertyChanged(string attribute)
		{
			switch (attribute)
			{
			case "posX":
				this._element.OnPropertyChanged("GridX");
				this._element.OnPropertyChanged("X");
				if (this._element.Parent != null)
				{
					this._element.Parent.MeasureSize();
					return;
				}
				return;
			case "posY":
				this._element.OnPropertyChanged("GridY");
				this._element.OnPropertyChanged("Y");
				if (this._element.Parent != null)
				{
					this._element.Parent.MeasureSize();
					return;
				}
				return;
			case "comment":
				this._element.OnPropertyChanged("Comment");
				return;
			case "title":
			case "text":
			{
				this._element.OnPropertyChanged("LocalString");
				ComponentType type = this._element.Type;
				if (type != ComponentType.Button)
				{
					return;
				}
				FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(this._element.Key).SpecificationInfo.FindNodeByName(this._element.Name);
				if (formSpecModel != null && formSpecModel.SpecAction != null)
				{
					formSpecModel.SpecAction.OnPropertyChanged("LocalString");
					return;
				}
				return;
			}
			case "gridWidth":
				this._element.OnPropertyChanged("Width");
				this._element.OnPropertyChanged("GridWidth");
				return;
			case "gridHeight":
				this._element.OnPropertyChanged("Height");
				this._element.OnPropertyChanged("GridHeight");
				return;
			case "rowHeight":
				this._element.OnPropertyChanged("RowHeight");
				this._element.MeasureSize();
				return;
			case "aggregate":
			case "aggregateName":
				this._element.OnPropertyChanged("IsEnabledAggregate");
				if (this._element.Parent != null)
				{
					this._element.Parent.UpdateAggregateStatus();
					return;
				}
				return;
			case "totalRows":
				this._element.OnPropertyChanged("TotalRows");
				return;
			}
			this._element.OnPropertyChanged("");
		}

		// Token: 0x040003E1 RID: 993
		private XmlElement _element;

		// Token: 0x040003E2 RID: 994
		private List<FormAttributesUndoRedoCommand.Property> _list;

		// Token: 0x02000125 RID: 293
		private class Property
		{
			// Token: 0x170002B2 RID: 690
			// (get) Token: 0x06000A44 RID: 2628 RVA: 0x0003365B File Offset: 0x0003185B
			// (set) Token: 0x06000A45 RID: 2629 RVA: 0x00033663 File Offset: 0x00031863
			public string Attribute { get; private set; }

			// Token: 0x170002B3 RID: 691
			// (get) Token: 0x06000A46 RID: 2630 RVA: 0x0003366C File Offset: 0x0003186C
			// (set) Token: 0x06000A47 RID: 2631 RVA: 0x00033674 File Offset: 0x00031874
			public string OldValue { get; private set; }

			// Token: 0x170002B4 RID: 692
			// (get) Token: 0x06000A48 RID: 2632 RVA: 0x0003367D File Offset: 0x0003187D
			// (set) Token: 0x06000A49 RID: 2633 RVA: 0x00033685 File Offset: 0x00031885
			public string NewValue { get; private set; }

			// Token: 0x06000A4A RID: 2634 RVA: 0x0003368E File Offset: 0x0003188E
			public Property(string att, string oldValue, string newValue)
			{
				this.Attribute = att;
				this.OldValue = oldValue;
				this.NewValue = newValue;
			}
		}
	}
}
