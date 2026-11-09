using System;
using System.Windows;

namespace SpecDesignerCommon.Exceptions
{
	// Token: 0x0200003B RID: 59
	public class NameAlreadyInUseException : Exception
	{
		// Token: 0x060001E6 RID: 486 RVA: 0x00008CA4 File Offset: 0x00006EA4
		public NameAlreadyInUseException(string name)
		{
			string text = Application.Current.Resources["Message_NameAlreadyInUse"] as string;
			this._message = string.Format(text, name);
		}

		// Token: 0x17000070 RID: 112
		// (get) Token: 0x060001E7 RID: 487 RVA: 0x00008CDE File Offset: 0x00006EDE
		public override string Message
		{
			get
			{
				return this._message;
			}
		}

		// Token: 0x040000B4 RID: 180
		private string _message;
	}
}
