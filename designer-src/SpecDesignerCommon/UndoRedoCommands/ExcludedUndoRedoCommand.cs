using System;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedoCommands
{
	// Token: 0x02000071 RID: 113
	public class ExcludedUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x06000462 RID: 1122 RVA: 0x00014008 File Offset: 0x00012208
		public ExcludedUndoRedoCommand(FormSpecModel model, bool isExcluded)
		{
			this._model = model;
			this._isExcluded = isExcluded;
		}

		// Token: 0x06000463 RID: 1123 RVA: 0x00014020 File Offset: 0x00012220
		public void Undo()
		{
			this._model.SetExcluded(!this._isExcluded);
			ComponentHelper.Get(this._model.Key).AddSelection(this._model.GeneroComponent, false);
			this._model.OnPropertyChanged("IsExcluded");
		}

		// Token: 0x06000464 RID: 1124 RVA: 0x00014074 File Offset: 0x00012274
		public void Execute()
		{
			this._model.SetExcluded(this._isExcluded);
			ComponentHelper.Get(this._model.Key).AddSelection(this._model.GeneroComponent, false);
			this._model.OnPropertyChanged("IsExcluded");
		}

		// Token: 0x06000465 RID: 1125 RVA: 0x000140C3 File Offset: 0x000122C3
		public void Clear()
		{
		}

		// Token: 0x040001B6 RID: 438
		private FormSpecModel _model;

		// Token: 0x040001B7 RID: 439
		private bool _isExcluded;
	}
}
