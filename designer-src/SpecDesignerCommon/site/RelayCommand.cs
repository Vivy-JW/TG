using System;
using System.Windows.Input;

namespace SpecDesignerCommon.Site
{
	// Token: 0x0200001C RID: 28
	public class RelayCommand : ICommand
	{
		// Token: 0x060000D1 RID: 209 RVA: 0x00005C29 File Offset: 0x00003E29
		public RelayCommand(Action<object> execute)
			: this(execute, null)
		{
		}

		// Token: 0x060000D2 RID: 210 RVA: 0x00005C33 File Offset: 0x00003E33
		public RelayCommand(Action<object> execute, Predicate<object> canExecute)
		{
			if (execute == null)
			{
				throw new ArgumentNullException("execute");
			}
			this._execute = execute;
			this._canExecute = canExecute;
		}

		// Token: 0x060000D3 RID: 211 RVA: 0x00005C57 File Offset: 0x00003E57
		public bool CanExecute(object parameter)
		{
			return this._canExecute == null || this._canExecute(parameter);
		}

		// Token: 0x14000002 RID: 2
		// (add) Token: 0x060000D4 RID: 212 RVA: 0x00005C6F File Offset: 0x00003E6F
		// (remove) Token: 0x060000D5 RID: 213 RVA: 0x00005C77 File Offset: 0x00003E77
		public event EventHandler CanExecuteChanged
		{
			add
			{
				CommandManager.RequerySuggested += value;
			}
			remove
			{
				CommandManager.RequerySuggested -= value;
			}
		}

		// Token: 0x060000D6 RID: 214 RVA: 0x00005C7F File Offset: 0x00003E7F
		public void Execute(object parameter)
		{
			this._execute(parameter);
		}

		// Token: 0x04000053 RID: 83
		private readonly Action<object> _execute;

		// Token: 0x04000054 RID: 84
		private readonly Predicate<object> _canExecute;
	}
}
