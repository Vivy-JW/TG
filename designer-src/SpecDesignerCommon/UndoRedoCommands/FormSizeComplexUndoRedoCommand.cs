using System;
using SpecDesignerCommon.UndoRedo;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedoCommands
{
	// Token: 0x020000E6 RID: 230
	public class FormSizeComplexUndoRedoCommand : GeneralComplexCommand
	{
		// Token: 0x060007BD RID: 1981 RVA: 0x000227BB File Offset: 0x000209BB
		public FormSizeComplexUndoRedoCommand(IUndoRedoCommand owner)
			: base(owner, true)
		{
		}

		// Token: 0x060007BE RID: 1982 RVA: 0x000227C5 File Offset: 0x000209C5
		public FormSizeComplexUndoRedoCommand(IUndoRedoCommand owner, bool isUndoReverse)
			: base(owner, isUndoReverse)
		{
		}

		// Token: 0x060007BF RID: 1983 RVA: 0x000227CF File Offset: 0x000209CF
		public override void Append(IUndoRedoCommand command)
		{
			if (command is FormSizeUndoRedoCommand || command is FormPosUndoRedoCommand)
			{
				base.Append(command);
			}
		}
	}
}
