using System;

namespace UndoRedoFramework.Commands
{
	// Token: 0x02000007 RID: 7
	public interface IGroupUndoRedoCommand
	{
		// Token: 0x06000023 RID: 35
		void Undo();

		// Token: 0x06000024 RID: 36
		void Execute();

		// Token: 0x06000025 RID: 37
		void Clear();

		// Token: 0x06000026 RID: 38
		void Append(IUndoRedoCommand subCommand);
	}
}
