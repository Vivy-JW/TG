using System;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x020000FF RID: 255
	public class ChangeProgRelProgramUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x06000845 RID: 2117 RVA: 0x0002447F File Offset: 0x0002267F
		public ChangeProgRelProgramUndoRedoCommand(SpecProgRelNode specNode, ProgRelProgram program, bool isDelete)
		{
			this._specNode = specNode;
			this._program = program;
			this._isDelete = isDelete;
		}

		// Token: 0x06000846 RID: 2118 RVA: 0x0002449C File Offset: 0x0002269C
		private void ModifyProgRelProgram(bool isDelete)
		{
			switch (isDelete)
			{
			case false:
				this._specNode.Appand(this._program);
				break;
			case true:
				this._specNode.Remove(this._program);
				break;
			}
			FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(this._specNode.ProgramKey).SpecificationInfo.FindNodeByName(this._specNode.Name);
			if (formSpecModel != null)
			{
				ComponentHelper.Get(this._specNode.ProgramKey).AddSelection(formSpecModel.GeneroComponent, false);
			}
		}

		// Token: 0x06000847 RID: 2119 RVA: 0x00024529 File Offset: 0x00022729
		public void Undo()
		{
			this.ModifyProgRelProgram(!this._isDelete);
		}

		// Token: 0x06000848 RID: 2120 RVA: 0x0002453A File Offset: 0x0002273A
		public void Execute()
		{
			this.ModifyProgRelProgram(this._isDelete);
		}

		// Token: 0x06000849 RID: 2121 RVA: 0x00024548 File Offset: 0x00022748
		public void Clear()
		{
		}

		// Token: 0x040002F1 RID: 753
		private SpecProgRelNode _specNode;

		// Token: 0x040002F2 RID: 754
		private ProgRelProgram _program;

		// Token: 0x040002F3 RID: 755
		private bool _isDelete;
	}
}
