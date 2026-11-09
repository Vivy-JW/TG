using System;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.UndoRedoCommands
{
	// Token: 0x02000017 RID: 23
	public class FormPosUndoRedoCommand : AbstractUndoRedoCommand
	{
		// Token: 0x060000B3 RID: 179 RVA: 0x00004CEC File Offset: 0x00002EEC
		public FormPosUndoRedoCommand(XmlElement element, string attribute, int newSize)
			: base(element)
		{
			this._attribute = attribute;
			this._newSize = newSize;
			string text;
			if ((text = attribute.ToLower()) != null)
			{
				if (text == "posx")
				{
					this._oldSize = element.GridX;
					this._changeType = FormPosUndoRedoCommand.ChangeType.GridX;
					return;
				}
				if (!(text == "posy"))
				{
					return;
				}
				this._oldSize = element.GridY;
				this._changeType = FormPosUndoRedoCommand.ChangeType.GridY;
			}
		}

		// Token: 0x060000B4 RID: 180 RVA: 0x00004D74 File Offset: 0x00002F74
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
			if (base.Element.Parent != null)
			{
				base.Element.Parent.SortNodes();
			}
			if (base.Element.Parent != null && this._newSize > this._oldSize)
			{
				base.Element.Parent.MeasureSize();
			}
			this.OnPropertyChanged();
			SettingManager.Get().GetUndoRedoManager(base.Element.Key).EndGroup(formSizeComplexUndoRedoCommand);
		}

		// Token: 0x060000B5 RID: 181 RVA: 0x00004E45 File Offset: 0x00003045
		public override void Undo()
		{
			if (-1 == this._oldSize)
			{
				return;
			}
			base.Element.SetAttribute(this._attribute, this._oldSize.ToString());
			this.OnPropertyChanged();
		}

		// Token: 0x060000B6 RID: 182 RVA: 0x00004E74 File Offset: 0x00003074
		private void OnPropertyChanged()
		{
			string attribute;
			if ((attribute = this._attribute) != null)
			{
				if (attribute == "posX")
				{
					base.Element.OnPropertyChanged("X");
					base.Element.OnPropertyChanged("GridX");
					return;
				}
				if (!(attribute == "posY"))
				{
					return;
				}
				base.Element.OnPropertyChanged("Y");
				base.Element.OnPropertyChanged("GridY");
			}
		}

		// Token: 0x060000B7 RID: 183 RVA: 0x00004EE8 File Offset: 0x000030E8
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

		// Token: 0x0400003B RID: 59
		private int _newSize = -1;

		// Token: 0x0400003C RID: 60
		private int _oldSize = -1;

		// Token: 0x0400003D RID: 61
		private string _attribute = string.Empty;

		// Token: 0x0400003E RID: 62
		private FormPosUndoRedoCommand.ChangeType _changeType;

		// Token: 0x02000018 RID: 24
		private enum ChangeType
		{
			// Token: 0x04000040 RID: 64
			GridX,
			// Token: 0x04000041 RID: 65
			GridY
		}
	}
}
