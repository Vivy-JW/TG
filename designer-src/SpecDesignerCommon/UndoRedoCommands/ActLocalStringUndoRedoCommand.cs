using System;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedoCommands
{
	// Token: 0x0200010F RID: 271
	public class ActLocalStringUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x06000962 RID: 2402 RVA: 0x0002FE0C File Offset: 0x0002E00C
		public ActLocalStringUndoRedoCommand(SpecActionNode actNode, string newValue)
		{
			this._actNode = actNode;
			this._oldValue = this.GetSpecificationInfo().GetActLocalStringText(this._actNode.Name);
			if (this._oldValue == null)
			{
				this._oldValue = this._actNode.Name;
			}
			this._newValue = newValue;
			FormSpecModel formSpecModel = this.GetSpecificationInfo().FindNodeByName(this._actNode.Name);
			this._form = ((formSpecModel == null) ? null : formSpecModel.GeneroComponent);
		}

		// Token: 0x06000963 RID: 2403 RVA: 0x0002FEA4 File Offset: 0x0002E0A4
		public void Undo()
		{
			this.GetSpecificationInfo().SetActLocalStringText(this._actNode.Name, this._oldValue);
			this._actNode.OnPropertyChanged("LocalString");
			if (this._form != null)
			{
				this._form.OnPropertyChanged("LocalString");
			}
		}

		// Token: 0x06000964 RID: 2404 RVA: 0x0002FEF8 File Offset: 0x0002E0F8
		public void Execute()
		{
			this.GetSpecificationInfo().SetActLocalStringText(this._actNode.Name, this._newValue);
			this._actNode.OnPropertyChanged("LocalString");
			this._actNode.OnPropertyChanged("Status");
			if (this._form != null)
			{
				this._form.OnPropertyChanged("LocalString");
			}
		}

		// Token: 0x06000965 RID: 2405 RVA: 0x0002FF59 File Offset: 0x0002E159
		public void Clear()
		{
			this._oldValue = string.Empty;
			this._newValue = string.Empty;
		}

		// Token: 0x06000966 RID: 2406 RVA: 0x0002FF71 File Offset: 0x0002E171
		private SpecificationInfo GetSpecificationInfo()
		{
			if (!(null == this._actNode.ProgramKey))
			{
				return SettingManager.Get().GetTzpManger(this._actNode.ProgramKey).SpecificationInfo;
			}
			return null;
		}

		// Token: 0x0400037F RID: 895
		private SpecActionNode _actNode;

		// Token: 0x04000380 RID: 896
		private XmlElement _form;

		// Token: 0x04000381 RID: 897
		private string _oldValue = string.Empty;

		// Token: 0x04000382 RID: 898
		private string _newValue = string.Empty;
	}
}
