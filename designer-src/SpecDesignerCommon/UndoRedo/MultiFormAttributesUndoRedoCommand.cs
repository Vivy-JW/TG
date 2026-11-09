using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000076 RID: 118
	public class MultiFormAttributesUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x0600048A RID: 1162 RVA: 0x000146BC File Offset: 0x000128BC
		public MultiFormAttributesUndoRedoCommand(IEnumerable<XmlElement> elements, string attribute, string newValue)
		{
			this._records = new Dictionary<XmlElement, List<MultiFormAttributesUndoRedoCommand.Property>>();
			foreach (XmlElement xmlElement in elements)
			{
				if (null == this._key)
				{
					this._key = xmlElement.Key;
				}
				List<MultiFormAttributesUndoRedoCommand.Property> list = new List<MultiFormAttributesUndoRedoCommand.Property>();
				list.Add(new MultiFormAttributesUndoRedoCommand.Property(attribute, xmlElement.GetAttribute(attribute), newValue));
				if (attribute != null)
				{
					if (!(attribute == "width"))
					{
						if (attribute == "height")
						{
							if (XmlElement.NOSET == newValue)
							{
								list.Add(new MultiFormAttributesUndoRedoCommand.Property("unitHeight", xmlElement.GetAttribute("unitHeight"), XmlElement.NOSET));
							}
						}
					}
					else if (XmlElement.NOSET == newValue)
					{
						list.Add(new MultiFormAttributesUndoRedoCommand.Property("unitWidth", xmlElement.GetAttribute("unitWidth"), XmlElement.NOSET));
					}
				}
				this._records.Add(xmlElement, list);
			}
		}

		// Token: 0x0600048B RID: 1163 RVA: 0x000147D4 File Offset: 0x000129D4
		public override void Execute()
		{
			if (this._records.Count == 0)
			{
				return;
			}
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(this._key).StartGroup(formSizeComplexUndoRedoCommand);
			foreach (KeyValuePair<XmlElement, List<MultiFormAttributesUndoRedoCommand.Property>> keyValuePair in this._records)
			{
				XmlElement key = keyValuePair.Key;
				foreach (MultiFormAttributesUndoRedoCommand.Property property in keyValuePair.Value)
				{
					string attribute;
					if ((attribute = property.Attribute) != null)
					{
						if (attribute == "rowCount" || attribute == "rowHeight" || attribute == "totalRows")
						{
							key.MeasureSize();
							continue;
						}
						if (attribute == "gridWidth")
						{
							key.GridWidth = int.Parse(property.NewValue);
							continue;
						}
						if (attribute == "gridHeight")
						{
							key.GridHeight = int.Parse(property.NewValue);
							continue;
						}
					}
					key.SetAttribute(property.Attribute, property.NewValue);
					this.OnPropertyChanged(key, property.Attribute);
				}
			}
			SettingManager.Get().GetUndoRedoManager(this._key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._key).MultipleSelection(this._records.Keys);
		}

		// Token: 0x0600048C RID: 1164 RVA: 0x00014970 File Offset: 0x00012B70
		public override void Undo()
		{
			if (this._records.Count == 0)
			{
				return;
			}
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = null;
			foreach (KeyValuePair<XmlElement, List<MultiFormAttributesUndoRedoCommand.Property>> keyValuePair in this._records)
			{
				XmlElement key = keyValuePair.Key;
				foreach (MultiFormAttributesUndoRedoCommand.Property property in keyValuePair.Value)
				{
					string attribute;
					if ((attribute = property.Attribute) != null && attribute == "totalRows")
					{
						formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
						SettingManager.Get().GetUndoRedoManager(this._key).StartGroup(formSizeComplexUndoRedoCommand);
						key.SetAttribute(property.Attribute, property.OldValue);
					}
					else
					{
						key.SetAttribute(property.Attribute, property.OldValue);
					}
					this.OnPropertyChanged(key, property.Attribute);
				}
			}
			if (formSizeComplexUndoRedoCommand != null)
			{
				SettingManager.Get().GetUndoRedoManager(this._key).EndGroup(formSizeComplexUndoRedoCommand);
			}
			ComponentHelper.Get(this._key).MultipleSelection(this._records.Keys);
		}

		// Token: 0x0600048D RID: 1165 RVA: 0x00014AB8 File Offset: 0x00012CB8
		public override void Clear()
		{
			if (this._records != null)
			{
				this._records.Clear();
			}
		}

		// Token: 0x0600048E RID: 1166 RVA: 0x00014AD0 File Offset: 0x00012CD0
		private void OnPropertyChanged(XmlElement element, string attribute)
		{
			switch (attribute)
			{
			case "posX":
				element.OnPropertyChanged("GridX");
				element.OnPropertyChanged("X");
				if (element.Parent != null)
				{
					element.Parent.MeasureSize();
					return;
				}
				return;
			case "posY":
				element.OnPropertyChanged("GridY");
				element.OnPropertyChanged("Y");
				if (element.Parent != null)
				{
					element.Parent.MeasureSize();
					return;
				}
				return;
			case "comment":
				element.OnPropertyChanged("Comment");
				return;
			case "title":
			case "text":
				element.OnPropertyChanged("LocalString");
				return;
			case "gridWidth":
				element.OnPropertyChanged("Width");
				element.OnPropertyChanged("GridWidth");
				return;
			case "gridHeight":
				element.OnPropertyChanged("Height");
				element.OnPropertyChanged("GridHeight");
				return;
			case "rowHeight":
				element.OnPropertyChanged("RowHeight");
				element.MeasureSize();
				return;
			case "aggregate":
				element.OnPropertyChanged("IsAnySiblingEnabledAggregate");
				return;
			case "totalRows":
				element.OnPropertyChanged("TotalRows");
				return;
			}
			element.OnPropertyChanged("");
		}

		// Token: 0x040001C6 RID: 454
		private Dictionary<XmlElement, List<MultiFormAttributesUndoRedoCommand.Property>> _records;

		// Token: 0x040001C7 RID: 455
		private PackageKey _key;

		// Token: 0x02000077 RID: 119
		private class Property
		{
			// Token: 0x17000139 RID: 313
			// (get) Token: 0x0600048F RID: 1167 RVA: 0x00014C94 File Offset: 0x00012E94
			// (set) Token: 0x06000490 RID: 1168 RVA: 0x00014C9C File Offset: 0x00012E9C
			public string Attribute { get; private set; }

			// Token: 0x1700013A RID: 314
			// (get) Token: 0x06000491 RID: 1169 RVA: 0x00014CA5 File Offset: 0x00012EA5
			// (set) Token: 0x06000492 RID: 1170 RVA: 0x00014CAD File Offset: 0x00012EAD
			public string OldValue { get; private set; }

			// Token: 0x1700013B RID: 315
			// (get) Token: 0x06000493 RID: 1171 RVA: 0x00014CB6 File Offset: 0x00012EB6
			// (set) Token: 0x06000494 RID: 1172 RVA: 0x00014CBE File Offset: 0x00012EBE
			public string NewValue { get; private set; }

			// Token: 0x06000495 RID: 1173 RVA: 0x00014CC7 File Offset: 0x00012EC7
			public Property(string att, string oldValue, string newValue)
			{
				this.Attribute = att;
				this.OldValue = oldValue;
				this.NewValue = newValue;
			}
		}
	}
}
