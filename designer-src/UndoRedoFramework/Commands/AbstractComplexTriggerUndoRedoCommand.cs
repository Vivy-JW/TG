using System;

namespace UndoRedoFramework.Commands
{
	// Token: 0x02000009 RID: 9
	public abstract class AbstractComplexTriggerUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x06000031 RID: 49
		public abstract void Undo();

		// Token: 0x06000032 RID: 50
		public abstract void Execute();

		// Token: 0x06000033 RID: 51
		public abstract void Clear();
	}
}
