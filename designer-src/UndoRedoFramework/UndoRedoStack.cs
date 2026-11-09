using System;
using System.Collections.Generic;
using UndoRedoFramework.Commands;

namespace UndoRedoFramework
{
	// Token: 0x02000005 RID: 5
	public class UndoRedoStack
	{
		// Token: 0x17000007 RID: 7
		public IUndoRedoCommand this[int index]
		{
			get
			{
				return this.items[index];
			}
		}

		// Token: 0x17000008 RID: 8
		// (get) Token: 0x06000019 RID: 25 RVA: 0x000024A8 File Offset: 0x000006A8
		public int Count
		{
			get
			{
				return this.items.Count;
			}
		}

		// Token: 0x0600001A RID: 26 RVA: 0x000024B5 File Offset: 0x000006B5
		public void Push(IUndoRedoCommand item)
		{
			this.items.Add(item);
		}

		// Token: 0x0600001B RID: 27 RVA: 0x000024C4 File Offset: 0x000006C4
		public IUndoRedoCommand Pop()
		{
			if (this.items.Count > 0)
			{
				IUndoRedoCommand undoRedoCommand = this.items[this.items.Count - 1];
				this.items.RemoveAt(this.items.Count - 1);
				return undoRedoCommand;
			}
			return null;
		}

		// Token: 0x0600001C RID: 28 RVA: 0x00002513 File Offset: 0x00000713
		public void Remove(int itemAtPosition)
		{
			this.items.RemoveAt(itemAtPosition);
		}

		// Token: 0x0600001D RID: 29 RVA: 0x00002521 File Offset: 0x00000721
		public void RemoveFirst()
		{
			if (this.items.Count > 0)
			{
				this.items.RemoveAt(0);
			}
		}

		// Token: 0x0600001E RID: 30 RVA: 0x00002540 File Offset: 0x00000740
		public void Clear()
		{
			foreach (IUndoRedoCommand undoRedoCommand in this.items)
			{
				undoRedoCommand.Clear();
			}
			this.items.Clear();
		}

		// Token: 0x0400000C RID: 12
		private List<IUndoRedoCommand> items = new List<IUndoRedoCommand>();
	}
}
