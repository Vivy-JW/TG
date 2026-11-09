using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Interactivity;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000043 RID: 67
	public class TextBoxEnterBehavior : Behavior<TextBox>
	{
		// Token: 0x060001B7 RID: 439 RVA: 0x0000C259 File Offset: 0x0000A459
		protected override void OnAttached()
		{
			base.AssociatedObject.AddHandler(UIElement.KeyUpEvent, new KeyEventHandler(this.EnterKeyUp), true);
			base.OnAttached();
		}

		// Token: 0x060001B8 RID: 440 RVA: 0x0000C280 File Offset: 0x0000A480
		public void EnterKeyUp(object sender, KeyEventArgs args)
		{
			if (args.Key == Key.Return)
			{
				BindingExpression bindingExpression = base.AssociatedObject.GetBindingExpression(TextBox.TextProperty);
				bindingExpression.UpdateSource();
			}
		}

		// Token: 0x060001B9 RID: 441 RVA: 0x0000C2AD File Offset: 0x0000A4AD
		protected override void OnDetaching()
		{
			base.OnDetaching();
			base.AssociatedObject.RemoveHandler(UIElement.KeyUpEvent, new KeyEventHandler(this.EnterKeyUp));
		}
	}
}
