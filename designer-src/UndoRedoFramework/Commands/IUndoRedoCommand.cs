using System;

namespace UndoRedoFramework.Commands
{
	// Token: 0x02000006 RID: 6
	public interface IUndoRedoCommand
	{
		// Token: 0x06000020 RID: 32
		void Undo();

		// Token: 0x06000021 RID: 33
		void Execute();

		// Token: 0x06000022 RID: 34
		void Clear();
	}
}
