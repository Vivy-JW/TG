using System;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x020000A9 RID: 169
	public class RenameUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x06000720 RID: 1824 RVA: 0x0001FC4E File Offset: 0x0001DE4E
		public RenameUndoRedoCommand(PackageKey key, string oldName, string newName)
		{
			this._key = key;
			this._newName = newName;
			this._oldName = oldName;
		}

		// Token: 0x06000721 RID: 1825 RVA: 0x0001FC84 File Offset: 0x0001DE84
		public void Undo()
		{
			this.GetSpecificationInfo().Rename(this._newName, this._oldName);
			FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(this._key).SpecificationInfo.FindNodeByName(this._oldName);
			if (formSpecModel != null && formSpecModel.GeneroComponent != null)
			{
				formSpecModel.GeneroComponent.OnPropertyChanged("Text");
				formSpecModel.GeneroComponent.OnPropertyChanged("Name");
				ComponentHelper.Get(this._key).AddSelection(formSpecModel.GeneroComponent, false);
				if (formSpecModel.SpecAction != null)
				{
					formSpecModel.SpecAction.OnPropertyChanged("Text");
				}
			}
		}

		// Token: 0x06000722 RID: 1826 RVA: 0x0001FD24 File Offset: 0x0001DF24
		public void Execute()
		{
			this.GetSpecificationInfo().Rename(this._oldName, this._newName);
			FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(this._key).SpecificationInfo.FindNodeByName(this._newName);
			if (formSpecModel != null && formSpecModel.GeneroComponent != null)
			{
				formSpecModel.GeneroComponent.OnPropertyChanged("Text");
				formSpecModel.GeneroComponent.OnPropertyChanged("Name");
				ComponentHelper.Get(this._key).AddSelection(formSpecModel.GeneroComponent, false);
				if (formSpecModel.SpecAction != null)
				{
					formSpecModel.SpecAction.OnPropertyChanged("LocalString");
				}
			}
		}

		// Token: 0x06000723 RID: 1827 RVA: 0x0001FDC3 File Offset: 0x0001DFC3
		public void Clear()
		{
		}

		// Token: 0x06000724 RID: 1828 RVA: 0x0001FDC5 File Offset: 0x0001DFC5
		private SpecificationInfo GetSpecificationInfo()
		{
			return SettingManager.Get().GetTzpManger(this._key).SpecificationInfo;
		}

		// Token: 0x0400028A RID: 650
		private string _newName = string.Empty;

		// Token: 0x0400028B RID: 651
		private string _oldName = string.Empty;

		// Token: 0x0400028C RID: 652
		private PackageKey _key;
	}
}
