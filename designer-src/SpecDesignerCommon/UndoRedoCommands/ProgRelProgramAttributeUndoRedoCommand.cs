using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedoCommands
{
	// Token: 0x02000005 RID: 5
	public class ProgRelProgramAttributeUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x06000034 RID: 52 RVA: 0x00002B20 File Offset: 0x00000D20
		public ProgRelProgramAttributeUndoRedoCommand(ProgRelProgram program, SpecProgRelNode progRelNode)
		{
			this._program = program;
			this._fsm = SettingManager.Get().GetTzpManger(progRelNode.ProgramKey).SpecificationInfo.FindNodeByName(progRelNode.Name);
		}

		// Token: 0x06000035 RID: 53 RVA: 0x00002B78 File Offset: 0x00000D78
		public void AddAttributeChanged(string attr, string oldValue, string newValue)
		{
			if (oldValue == newValue)
			{
				return;
			}
			if (!this._oldAttributes.ContainsKey(attr))
			{
				this._oldAttributes.Add(attr, oldValue);
			}
			if (!this._newAttributes.ContainsKey(attr))
			{
				this._newAttributes.Add(attr, newValue);
			}
		}

		// Token: 0x06000036 RID: 54 RVA: 0x00002BC8 File Offset: 0x00000DC8
		public void Undo()
		{
			foreach (KeyValuePair<string, string> keyValuePair in this._oldAttributes)
			{
				this._program.SetAttribute(keyValuePair.Key, keyValuePair.Value);
			}
			this.AfterPropertyChanged();
		}

		// Token: 0x06000037 RID: 55 RVA: 0x00002C34 File Offset: 0x00000E34
		public void Execute()
		{
			foreach (KeyValuePair<string, string> keyValuePair in this._newAttributes)
			{
				this._program.SetAttribute(keyValuePair.Key, keyValuePair.Value);
			}
			this.AfterPropertyChanged();
		}

		// Token: 0x06000038 RID: 56 RVA: 0x00002CA0 File Offset: 0x00000EA0
		private void AfterPropertyChanged()
		{
			this._program.OnPropertyChanged("");
			if (this._fsm.SpecField != null)
			{
				this._fsm.SpecField.OnPropertyChanged("Status");
			}
			ComponentHelper.Get(this._fsm.Key).AddSelection(this._fsm.GeneroComponent, false);
		}

		// Token: 0x06000039 RID: 57 RVA: 0x00002D00 File Offset: 0x00000F00
		public void Clear()
		{
			this._newAttributes.Clear();
			this._oldAttributes.Clear();
		}

		// Token: 0x04000011 RID: 17
		private ProgRelProgram _program;

		// Token: 0x04000012 RID: 18
		private Dictionary<string, string> _oldAttributes = new Dictionary<string, string>();

		// Token: 0x04000013 RID: 19
		private Dictionary<string, string> _newAttributes = new Dictionary<string, string>();

		// Token: 0x04000014 RID: 20
		private FormSpecModel _fsm;
	}
}
