using System;
using UndoRedoFramework.Commands;

namespace UndoRedoFramework
{
	// Token: 0x02000004 RID: 4
	public class UndoRedoManager
	{
		// Token: 0x14000001 RID: 1
		// (add) Token: 0x06000008 RID: 8 RVA: 0x000020D0 File Offset: 0x000002D0
		// (remove) Token: 0x06000009 RID: 9 RVA: 0x00002108 File Offset: 0x00000308
		public event EventHandler UndoStackChanged;

		// Token: 0x17000004 RID: 4
		// (get) Token: 0x0600000A RID: 10 RVA: 0x0000213D File Offset: 0x0000033D
		// (set) Token: 0x0600000B RID: 11 RVA: 0x00002145 File Offset: 0x00000345
		public int MAXSTACKSIZE { get; private set; }

		// Token: 0x0600000C RID: 12 RVA: 0x0000214E File Offset: 0x0000034E
		public UndoRedoManager(int maxStackSize)
		{
			this.MAXSTACKSIZE = maxStackSize;
		}

		// Token: 0x0600000D RID: 13 RVA: 0x0000215D File Offset: 0x0000035D
		public void Init()
		{
			this._UndoStack = new UndoRedoStack();
			this._RedoStack = new UndoRedoStack();
		}

		// Token: 0x0600000E RID: 14 RVA: 0x00002175 File Offset: 0x00000375
		public void StartGroup(GeneralComplexCommand complexCommand)
		{
			if (this._isStartGroup || this._isUndoing)
			{
				return;
			}
			this._isStartGroup = true;
			this._groupCommand = complexCommand;
		}

		// Token: 0x0600000F RID: 15 RVA: 0x00002198 File Offset: 0x00000398
		public void EndGroup(GeneralComplexCommand complexCommand)
		{
			if (!this._isStartGroup || this._isUndoing)
			{
				return;
			}
			if (this._groupCommand == null)
			{
				return;
			}
			if (this._groupCommand != complexCommand)
			{
				foreach (IUndoRedoCommand undoRedoCommand in complexCommand.GetCommands())
				{
					this._groupCommand.Append(undoRedoCommand);
				}
				return;
			}
			if (this._UndoStack == null)
			{
				this.Init();
			}
			this._isStartGroup = false;
			if (this._groupCommand != null && this._groupCommand.CommandCount() > 0)
			{
				if (!this._isRedoing && this._RedoStack != null)
				{
					this._RedoStack.Clear();
				}
				this._isRedoing = false;
				this.AdjustUndoStack();
				this._UndoStack.Push(this._groupCommand);
			}
			this._groupCommand = null;
		}

		// Token: 0x06000010 RID: 16 RVA: 0x0000227C File Offset: 0x0000047C
		public void Undo()
		{
			if (this._UndoStack == null)
			{
				this.Init();
			}
			if (this._UndoStack.Count == 0)
			{
				return;
			}
			this._isUndoing = true;
			IUndoRedoCommand undoRedoCommand = this._UndoStack.Pop();
			undoRedoCommand.Undo();
			if (undoRedoCommand is GeneralComplexCommand)
			{
				GeneralComplexCommand generalComplexCommand = undoRedoCommand as GeneralComplexCommand;
				this._RedoStack.Push(generalComplexCommand.MainCommand);
			}
			else
			{
				this._RedoStack.Push(undoRedoCommand);
			}
			this._isUndoing = false;
			if (this.UndoStackChanged != null)
			{
				this.UndoStackChanged(this, EventArgs.Empty);
			}
		}

		// Token: 0x06000011 RID: 17 RVA: 0x0000230C File Offset: 0x0000050C
		public void Redo()
		{
			if (this._RedoStack == null)
			{
				this.Init();
			}
			if (this._RedoStack.Count == 0)
			{
				return;
			}
			this._isRedoing = true;
			IUndoRedoCommand undoRedoCommand = this._RedoStack.Pop();
			undoRedoCommand.Execute();
			if (!(undoRedoCommand is AbstractComplexTriggerUndoRedoCommand))
			{
				this._UndoStack.Push(undoRedoCommand);
			}
			if (this.UndoStackChanged != null)
			{
				this.UndoStackChanged(this, EventArgs.Empty);
			}
		}

		// Token: 0x06000012 RID: 18 RVA: 0x0000237B File Offset: 0x0000057B
		public void AddThenExecute(IUndoRedoCommand command)
		{
			command.Execute();
			this.AddUndo(command);
		}

		// Token: 0x06000013 RID: 19 RVA: 0x0000238C File Offset: 0x0000058C
		private void AdjustUndoStack()
		{
			if (this.MAXSTACKSIZE != -1 && this._UndoStack.Count >= this.MAXSTACKSIZE)
			{
				IUndoRedoCommand undoRedoCommand = this._UndoStack[0];
				undoRedoCommand.Clear();
				this._UndoStack.RemoveFirst();
			}
		}

		// Token: 0x06000014 RID: 20 RVA: 0x000023D4 File Offset: 0x000005D4
		public void AddUndo(IUndoRedoCommand command)
		{
			if (this._isUndoing)
			{
				return;
			}
			if (this._UndoStack == null)
			{
				this.Init();
			}
			this._RedoStack.Clear();
			if (!this._isStartGroup)
			{
				this.AdjustUndoStack();
				this._UndoStack.Push(command);
			}
			else
			{
				this._groupCommand.Append(command);
			}
			if (this.UndoStackChanged != null)
			{
				this.UndoStackChanged(this, EventArgs.Empty);
			}
		}

		// Token: 0x17000005 RID: 5
		// (get) Token: 0x06000015 RID: 21 RVA: 0x00002444 File Offset: 0x00000644
		public int UndoCount
		{
			get
			{
				if (this._UndoStack == null)
				{
					return 0;
				}
				return this._UndoStack.Count;
			}
		}

		// Token: 0x17000006 RID: 6
		// (get) Token: 0x06000016 RID: 22 RVA: 0x0000245B File Offset: 0x0000065B
		public int RedoCount
		{
			get
			{
				if (this._RedoStack == null)
				{
					return 0;
				}
				return this._RedoStack.Count;
			}
		}

		// Token: 0x06000017 RID: 23 RVA: 0x00002472 File Offset: 0x00000672
		public void Clear()
		{
			if (this._UndoStack != null)
			{
				this._UndoStack.Clear();
			}
			if (this._RedoStack != null)
			{
				this._RedoStack.Clear();
			}
		}

		// Token: 0x04000004 RID: 4
		private bool _isStartGroup;

		// Token: 0x04000005 RID: 5
		private GeneralComplexCommand _groupCommand;

		// Token: 0x04000006 RID: 6
		private bool _isRedoing;

		// Token: 0x04000007 RID: 7
		private bool _isUndoing;

		// Token: 0x04000009 RID: 9
		private UndoRedoStack _UndoStack;

		// Token: 0x0400000A RID: 10
		private UndoRedoStack _RedoStack;
	}
}
