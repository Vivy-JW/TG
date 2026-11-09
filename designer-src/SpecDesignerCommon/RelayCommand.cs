using System;
using System.Windows.Input;

namespace SpecDesignerCommon
{
	// Token: 0x020000E9 RID: 233
	public class RelayCommand : ICommand
	{
		// Token: 0x060007D3 RID: 2003 RVA: 0x0002301A File Offset: 0x0002121A
		public RelayCommand(Action<object> execute)
			: this(execute, null)
		{
		}

		// Token: 0x060007D4 RID: 2004 RVA: 0x00023024 File Offset: 0x00021224
		public RelayCommand(Action<object> execute, Predicate<object> canExecute)
		{
			if (execute == null)
			{
				throw new ArgumentNullException("execute");
			}
			this._execute = execute;
			this._canExecute = canExecute;
		}

		// Token: 0x060007D5 RID: 2005 RVA: 0x00023048 File Offset: 0x00021248
		public bool CanExecute(object parameter)
		{
			return this._canExecute == null || this._canExecute(parameter);
		}

		// Token: 0x1400001D RID: 29
		// (add) Token: 0x060007D6 RID: 2006 RVA: 0x00023060 File Offset: 0x00021260
		// (remove) Token: 0x060007D7 RID: 2007 RVA: 0x00023068 File Offset: 0x00021268
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

		// Token: 0x060007D8 RID: 2008 RVA: 0x00023070 File Offset: 0x00021270
		public void Execute(object parameter)
		{
			this._execute(parameter);
		}

		// Token: 0x040002C5 RID: 709
		private readonly Action<object> _execute;

		// Token: 0x040002C6 RID: 710
		private readonly Predicate<object> _canExecute;
	}
}
