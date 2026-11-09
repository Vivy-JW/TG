using System;
using System.Collections.Generic;
using System.Linq;

namespace UndoRedoFramework.Commands
{
	// Token: 0x02000008 RID: 8
	public class GeneralComplexCommand : IUndoRedoCommand
	{
		// Token: 0x17000009 RID: 9
		// (get) Token: 0x06000027 RID: 39 RVA: 0x000025B3 File Offset: 0x000007B3
		public IUndoRedoCommand MainCommand
		{
			get
			{
				return this._mainCommand;
			}
		}

		// Token: 0x06000028 RID: 40 RVA: 0x000025BB File Offset: 0x000007BB
		public GeneralComplexCommand()
		{
		}

		// Token: 0x06000029 RID: 41 RVA: 0x000025CA File Offset: 0x000007CA
		public GeneralComplexCommand(IUndoRedoCommand mainCommand)
			: this(mainCommand, true)
		{
		}

		// Token: 0x0600002A RID: 42 RVA: 0x000025D4 File Offset: 0x000007D4
		public GeneralComplexCommand(IUndoRedoCommand mainCommand, bool isUndoReverse)
		{
			this._isUndoReverse = isUndoReverse;
			this._mainCommand = mainCommand;
			this._commands = new List<IUndoRedoCommand>();
			this._commands.Add(this._mainCommand);
		}

		// Token: 0x0600002B RID: 43 RVA: 0x0000260D File Offset: 0x0000080D
		public virtual void Append(IUndoRedoCommand command)
		{
			this._commands.Insert(0, command);
		}

		// Token: 0x0600002C RID: 44 RVA: 0x0000261C File Offset: 0x0000081C
		public virtual void Undo()
		{
			if (this._commands == null)
			{
				return;
			}
			switch (this._isUndoReverse)
			{
			case false:
			{
				for (int i = 0; i < this._commands.Count; i++)
				{
					this._commands[i].Undo();
				}
				return;
			}
			case true:
			{
				for (int j = this._commands.Count - 1; j >= 0; j--)
				{
					this._commands[j].Undo();
				}
				return;
			}
			default:
				return;
			}
		}

		// Token: 0x0600002D RID: 45 RVA: 0x00002699 File Offset: 0x00000899
		public virtual void Execute()
		{
			if (this._mainCommand == null)
			{
				return;
			}
			this._mainCommand.Execute();
		}

		// Token: 0x0600002E RID: 46 RVA: 0x000026B0 File Offset: 0x000008B0
		public virtual void Clear()
		{
			if (this._commands != null)
			{
				foreach (IUndoRedoCommand undoRedoCommand in this._commands)
				{
					undoRedoCommand.Clear();
				}
			}
		}

		// Token: 0x0600002F RID: 47 RVA: 0x0000270C File Offset: 0x0000090C
		public int CommandCount()
		{
			if (this._commands != null)
			{
				return this._commands.Count<IUndoRedoCommand>();
			}
			return 0;
		}

		// Token: 0x06000030 RID: 48 RVA: 0x000029A4 File Offset: 0x00000BA4
		public IEnumerable<IUndoRedoCommand> GetCommands()
		{
			foreach (IUndoRedoCommand cmd in this._commands)
			{
				if (cmd is GeneralComplexCommand)
				{
					GeneralComplexCommand childComplex = cmd as GeneralComplexCommand;
					foreach (IUndoRedoCommand subCmd in childComplex.GetCommands())
					{
						yield return subCmd;
					}
				}
				else
				{
					yield return cmd;
				}
			}
			yield break;
		}

		// Token: 0x0400000D RID: 13
		private bool _isUndoReverse = true;

		// Token: 0x0400000E RID: 14
		private IUndoRedoCommand _mainCommand;

		// Token: 0x0400000F RID: 15
		private List<IUndoRedoCommand> _commands;
	}
}
