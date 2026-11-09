using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x0200010D RID: 269
	public class FormSizeUndoRedoCommand : AbstractUndoRedoCommand
	{
		// Token: 0x0600095D RID: 2397 RVA: 0x0002F738 File Offset: 0x0002D938
		public FormSizeUndoRedoCommand(XmlElement element, string attribute, int newSize)
			: base(element)
		{
			this._attribute = attribute;
			this._newSize = newSize;
			string text;
			if ((text = attribute.ToLower()) != null)
			{
				if (text == "gridwidth")
				{
					this._oldSize = element.GridWidth;
					this._changeType = FormSizeUndoRedoCommand.ChangeType.Width;
					return;
				}
				if (!(text == "gridheight"))
				{
					return;
				}
				this._oldSize = element.GridHeight;
				this._changeType = FormSizeUndoRedoCommand.ChangeType.Height;
			}
		}

		// Token: 0x0600095E RID: 2398 RVA: 0x0002F7C0 File Offset: 0x0002D9C0
		public override void Execute()
		{
			if (-1 == this._oldSize)
			{
				return;
			}
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(base.Element.Key).StartGroup(formSizeComplexUndoRedoCommand);
			if (base.Element.GetAttribute(this._attribute) != null)
			{
				base.Element.SetAttribute(this._attribute, this._newSize.ToString());
			}
			if (this._changeType == FormSizeUndoRedoCommand.ChangeType.Height)
			{
				ComponentType type = base.Element.Type;
				if (type != ComponentType.HBox)
				{
					if (type == ComponentType.VBox && base.Element.Nodes.Count > 0)
					{
						int num = 0;
						for (int i = 0; i < base.Element.Nodes.Count; i++)
						{
							if (i < base.Element.Nodes.Count - 1)
							{
								num += base.Element.Nodes[i].GridHeight;
							}
						}
						base.Element.Nodes[base.Element.Nodes.Count - 1].GridHeight = this._newSize - num - 2;
					}
				}
				else
				{
					foreach (XmlElement xmlElement in base.Element.Nodes)
					{
						xmlElement.GridHeight = this._newSize;
					}
				}
				if (this._newSize > this._oldSize && base.Element.Parent != null)
				{
					base.Element.Parent.MeasureSize();
				}
				else if (this._newSize < this._oldSize)
				{
					switch (base.Element.Type)
					{
					case ComponentType.Folder:
					{
						int num2 = this._newSize - 1;
						foreach (XmlElement xmlElement2 in base.Element.Nodes)
						{
							xmlElement2.GridHeight = num2;
						}
						break;
					}
					}
				}
				if (base.Element.Parent != null)
				{
					ComponentType type2 = base.Element.Parent.Type;
					if (type2 == ComponentType.VBox)
					{
						base.Element.Parent.MeasureSize();
					}
				}
			}
			else
			{
				ComponentType type3 = base.Element.Type;
				if (type3 != ComponentType.HBox)
				{
					if (type3 == ComponentType.VBox)
					{
						foreach (XmlElement xmlElement3 in base.Element.Nodes)
						{
							xmlElement3.GridWidth = this._newSize;
						}
					}
				}
				else if (base.Element.Nodes.Count > 0)
				{
					int num3 = 0;
					for (int j = 0; j < base.Element.Nodes.Count; j++)
					{
						if (j < base.Element.Nodes.Count - 1)
						{
							num3 += base.Element.Nodes[j].GridWidth;
						}
					}
					base.Element.Nodes[base.Element.Nodes.Count - 1].GridWidth = this._newSize - num3 - 4;
				}
				if (base.Element.Parent != null)
				{
					XmlElement parent = base.Element.Parent;
					switch (parent.Type)
					{
					case ComponentType.HBox:
					case ComponentType.Table:
					case ComponentType.Tree:
					case ComponentType.VBox:
						parent.MeasurePos();
						break;
					}
				}
				if (this._newSize > this._oldSize && base.Element.Parent != null)
				{
					base.Element.Parent.MeasureSize();
				}
				else if (this._newSize < this._oldSize)
				{
					switch (base.Element.Type)
					{
					case ComponentType.Folder:
					{
						using (IEnumerator<XmlElement> enumerator4 = base.Element.Nodes.GetEnumerator())
						{
							while (enumerator4.MoveNext())
							{
								XmlElement xmlElement4 = enumerator4.Current;
								xmlElement4.GridWidth = this._newSize;
							}
							goto IL_04BF;
						}
						break;
					}
					case ComponentType.Form:
					case ComponentType.Grid:
					case ComponentType.Group:
					case ComponentType.HBox:
					case ComponentType.RadioGroup:
					case ComponentType.ScrollGrid:
					case ComponentType.Table:
					case ComponentType.Tree:
						goto IL_04BF;
					case ComponentType.Page:
						break;
					default:
						goto IL_04BF;
					}
					foreach (XmlElement xmlElement5 in base.Element.Nodes)
					{
						xmlElement5.GridWidth = base.Element.Parent.GridWidth;
					}
				}
			}
			IL_04BF:
			this.OnPropertyChanged();
			SettingManager.Get().GetUndoRedoManager(base.Element.Key).EndGroup(formSizeComplexUndoRedoCommand);
		}

		// Token: 0x0600095F RID: 2399 RVA: 0x0002FCF0 File Offset: 0x0002DEF0
		public override void Undo()
		{
			if (-1 == this._oldSize)
			{
				return;
			}
			base.Element.SetAttribute(this._attribute, this._oldSize.ToString());
			this.OnPropertyChanged();
		}

		// Token: 0x06000960 RID: 2400 RVA: 0x0002FD20 File Offset: 0x0002DF20
		private void OnPropertyChanged()
		{
			string attribute;
			if ((attribute = this._attribute) != null)
			{
				if (attribute == "gridWidth")
				{
					base.Element.OnPropertyChanged("");
					base.Element.OnPropertyChanged("Width");
					base.Element.OnPropertyChanged("GridWidth");
					return;
				}
				if (!(attribute == "gridHeight"))
				{
					return;
				}
				base.Element.OnPropertyChanged("");
				base.Element.OnPropertyChanged("Height");
				base.Element.OnPropertyChanged("GridHeight");
			}
		}

		// Token: 0x06000961 RID: 2401 RVA: 0x0002FDB4 File Offset: 0x0002DFB4
		public override string ToString()
		{
			return string.Format("{0}.{1}: {2} -> {3}", new object[]
			{
				base.Element.Name,
				this._changeType,
				this._oldSize,
				this._newSize
			});
		}

		// Token: 0x04000378 RID: 888
		private int _newSize = -1;

		// Token: 0x04000379 RID: 889
		private int _oldSize = -1;

		// Token: 0x0400037A RID: 890
		private string _attribute = string.Empty;

		// Token: 0x0400037B RID: 891
		private FormSizeUndoRedoCommand.ChangeType _changeType;

		// Token: 0x0200010E RID: 270
		private enum ChangeType
		{
			// Token: 0x0400037D RID: 893
			Height,
			// Token: 0x0400037E RID: 894
			Width
		}
	}
}
