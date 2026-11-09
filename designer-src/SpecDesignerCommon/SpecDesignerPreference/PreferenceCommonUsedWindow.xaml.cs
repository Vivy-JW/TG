using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq.Expressions;
using System.Runtime.CompilerServices;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using Microsoft.CSharp.RuntimeBinder;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon;

namespace SpecDesignerPreference
{
	// Token: 0x02000143 RID: 323
	public partial class PreferenceCommonUsedWindow : UserControl
	{
		// Token: 0x06000B40 RID: 2880 RVA: 0x0003666C File Offset: 0x0003486C
		public PreferenceCommonUsedWindow()
		{
			this.InitializeComponent();
			this.CommonSection = PreferenceManager.Current.Settings.CommonUsedSetting;
			Style style = new Style(typeof(ListBoxItem));
			style.Setters.Add(new Setter(UIElement.AllowDropProperty, true));
			style.Setters.Add(new EventSetter(UIElement.PreviewMouseLeftButtonDownEvent, new MouseButtonEventHandler(this.s_PreviewMouseLeftButtonDown)));
			style.Setters.Add(new EventSetter(UIElement.DragOverEvent, new DragEventHandler(this.Disable_DragOver)));
			this.Disable.ItemContainerStyle = style;
			Style style2 = new Style(typeof(ListBoxItem));
			style2.Setters.Add(new Setter(UIElement.AllowDropProperty, true));
			style2.Setters.Add(new EventSetter(UIElement.PreviewMouseLeftButtonDownEvent, new MouseButtonEventHandler(this.s_PreviewMouseLeftButtonDown)));
			style2.Setters.Add(new EventSetter(UIElement.DragOverEvent, new DragEventHandler(this.Enable_DragOver)));
			this.Enable.ItemContainerStyle = style2;
			base.CommandBindings.Add(new CommandBinding(PreferenceCommands.ShowProgramNoWindowCommand, new ExecutedRoutedEventHandler(PreferenceCommands.ExecutedProgramNoWindow), new CanExecuteRoutedEventHandler(PreferenceCommands.CanProgramNoWindow)));
		}

		// Token: 0x06000B41 RID: 2881 RVA: 0x000367BC File Offset: 0x000349BC
		private void AddButton_Click(object sender, RoutedEventArgs e)
		{
			CommonUsedShortcut commonUsedShortcut = new CommonUsedShortcut(this.ProgramName.Text, (this.Disable.Items.Count + 1).ToString(), "N", this.ProgramDesc.Text);
			if (string.IsNullOrEmpty(this.ProgramName.Text))
			{
				MessageBox.Show(Application.Current.FindResource("Preference_MissingProg") as string);
				return;
			}
			foreach (CommonUsedShortcut commonUsedShortcut2 in this.CommonSection.Items)
			{
				if (commonUsedShortcut2.Name == this.ProgramName.Text)
				{
					MessageBox.Show(Application.Current.FindResource("Preference_ProgExit") as string);
					return;
				}
			}
			this.CommonSection.Add(commonUsedShortcut);
			this.ProgramName.Text = "";
			this.ProgramDesc.Text = "";
		}

		// Token: 0x06000B42 RID: 2882 RVA: 0x000368D0 File Offset: 0x00034AD0
		private void EnableButton_Click(object sender, RoutedEventArgs e)
		{
			if (this.Disable.SelectedIndex == -1)
			{
				return;
			}
			object selectedItem = this.Disable.SelectedItem;
			foreach (CommonUsedShortcut commonUsedShortcut in this.CommonSection.Items)
			{
				if (PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site1 == null)
				{
					PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site1 = CallSite<Func<CallSite, object, bool>>.Create(Binder.UnaryOperation(CSharpBinderFlags.None, ExpressionType.IsTrue, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
				}
				Func<CallSite, object, bool> target = PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site1.Target;
				CallSite <>p__Site = PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site1;
				if (PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site2 == null)
				{
					PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site2 = CallSite<Func<CallSite, string, object, object>>.Create(Binder.BinaryOperation(CSharpBinderFlags.None, ExpressionType.Equal, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[]
					{
						CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType, null),
						CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null)
					}));
				}
				Func<CallSite, string, object, object> target2 = PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site2.Target;
				CallSite <>p__Site2 = PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site2;
				string name = commonUsedShortcut.Name;
				if (PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site3 == null)
				{
					PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site3 = CallSite<Func<CallSite, object, object>>.Create(Binder.GetMember(CSharpBinderFlags.None, "Name", typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
				}
				if (target(<>p__Site, target2(<>p__Site2, name, PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site3.Target(PreferenceCommonUsedWindow.<EnableButton_Click>o__SiteContainer0.<>p__Site3, selectedItem))))
				{
					this.CommonSection.Remove(commonUsedShortcut);
					commonUsedShortcut.Enabled = "Y";
					commonUsedShortcut.Sort = (this.Enable.Items.Count + 1).ToString();
					this.CommonSection.Add(commonUsedShortcut);
					break;
				}
			}
			this.ReSort();
		}

		// Token: 0x06000B43 RID: 2883 RVA: 0x00036A8C File Offset: 0x00034C8C
		private void ReSort()
		{
			int num = 1;
			foreach (CommonUsedShortcut commonUsedShortcut in this.CommonSection.EnableList)
			{
				commonUsedShortcut.Sort = num++.ToString();
			}
			num = 1;
			foreach (CommonUsedShortcut commonUsedShortcut2 in this.CommonSection.DisableList)
			{
				commonUsedShortcut2.Sort = num++.ToString();
			}
		}

		// Token: 0x06000B44 RID: 2884 RVA: 0x00036B44 File Offset: 0x00034D44
		private void DisableButton_Click(object sender, RoutedEventArgs e)
		{
			if (this.Enable.SelectedIndex == -1)
			{
				return;
			}
			object selectedItem = this.Enable.SelectedItem;
			foreach (CommonUsedShortcut commonUsedShortcut in this.CommonSection.DisableList)
			{
				if (PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site5 == null)
				{
					PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site5 = CallSite<Func<CallSite, object, bool>>.Create(Binder.UnaryOperation(CSharpBinderFlags.None, ExpressionType.IsTrue, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
				}
				Func<CallSite, object, bool> target = PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site5.Target;
				CallSite <>p__Site = PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site5;
				if (PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site6 == null)
				{
					PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site6 = CallSite<Func<CallSite, string, object, object>>.Create(Binder.BinaryOperation(CSharpBinderFlags.None, ExpressionType.Equal, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[]
					{
						CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType, null),
						CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null)
					}));
				}
				Func<CallSite, string, object, object> target2 = PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site6.Target;
				CallSite <>p__Site2 = PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site6;
				string name = commonUsedShortcut.Name;
				if (PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site7 == null)
				{
					PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site7 = CallSite<Func<CallSite, object, object>>.Create(Binder.GetMember(CSharpBinderFlags.None, "Name", typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
				}
				if (target(<>p__Site, target2(<>p__Site2, name, PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site7.Target(PreferenceCommonUsedWindow.<DisableButton_Click>o__SiteContainer4.<>p__Site7, selectedItem))))
				{
					this.CommonSection.Remove(commonUsedShortcut);
					commonUsedShortcut.Enabled = "N";
					commonUsedShortcut.Sort = (this.Disable.Items.Count + 1).ToString();
					this.CommonSection.Add(commonUsedShortcut);
					break;
				}
			}
			this.ReSort();
		}

		// Token: 0x06000B45 RID: 2885 RVA: 0x00036D00 File Offset: 0x00034F00
		private void DisableMenuItemDelete_Click(object sender, RoutedEventArgs e)
		{
			if (this.Disable.SelectedIndex == -1)
			{
				return;
			}
			object selectedItem = this.Disable.SelectedItem;
			if (PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Site9 == null)
			{
				PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Site9 = CallSite<Func<CallSite, object, bool>>.Create(Binder.UnaryOperation(CSharpBinderFlags.None, ExpressionType.IsTrue, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
			}
			Func<CallSite, object, bool> target = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Site9.Target;
			CallSite <>p__Site = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Site9;
			if (PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitea == null)
			{
				PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitea = CallSite<Func<CallSite, MessageBoxResult, object, object>>.Create(Binder.BinaryOperation(CSharpBinderFlags.None, ExpressionType.Equal, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[]
				{
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType | CSharpArgumentInfoFlags.Constant, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null)
				}));
			}
			Func<CallSite, MessageBoxResult, object, object> target2 = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitea.Target;
			CallSite <>p__Sitea = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitea;
			MessageBoxResult messageBoxResult = MessageBoxResult.Yes;
			if (PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Siteb == null)
			{
				PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Siteb = CallSite<Func<CallSite, Type, object, string, MessageBoxButton, MessageBoxImage, object>>.Create(Binder.InvokeMember(CSharpBinderFlags.None, "Show", null, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[]
				{
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType | CSharpArgumentInfoFlags.IsStaticType, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType | CSharpArgumentInfoFlags.Constant, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType | CSharpArgumentInfoFlags.Constant, null)
				}));
			}
			Func<CallSite, Type, object, string, MessageBoxButton, MessageBoxImage, object> target3 = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Siteb.Target;
			CallSite <>p__Siteb = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Siteb;
			Type typeFromHandle = typeof(DesignerMessageBox);
			if (PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitec == null)
			{
				PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitec = CallSite<Func<CallSite, Type, string, object, object>>.Create(Binder.InvokeMember(CSharpBinderFlags.None, "Format", null, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[]
				{
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType | CSharpArgumentInfoFlags.IsStaticType, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null)
				}));
			}
			Func<CallSite, Type, string, object, object> target4 = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitec.Target;
			CallSite <>p__Sitec = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitec;
			Type typeFromHandle2 = typeof(string);
			string text = Application.Current.FindResource("Preference_ConfirmDeleteProg") as string;
			if (PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sited == null)
			{
				PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sited = CallSite<Func<CallSite, object, object>>.Create(Binder.GetMember(CSharpBinderFlags.None, "Name", typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
			}
			if (target(<>p__Site, target2(<>p__Sitea, messageBoxResult, target3(<>p__Siteb, typeFromHandle, target4(<>p__Sitec, typeFromHandle2, text, PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sited.Target(PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sited, selectedItem)), Application.Current.FindResource("Preference_ConfirmDelete") as string, MessageBoxButton.YesNo, MessageBoxImage.Exclamation))))
			{
				foreach (CommonUsedShortcut commonUsedShortcut in this.CommonSection.Items)
				{
					if (PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitee == null)
					{
						PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitee = CallSite<Func<CallSite, object, bool>>.Create(Binder.UnaryOperation(CSharpBinderFlags.None, ExpressionType.IsTrue, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
					}
					Func<CallSite, object, bool> target5 = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitee.Target;
					CallSite <>p__Sitee = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitee;
					if (PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitef == null)
					{
						PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitef = CallSite<Func<CallSite, string, object, object>>.Create(Binder.BinaryOperation(CSharpBinderFlags.None, ExpressionType.Equal, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[]
						{
							CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType, null),
							CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null)
						}));
					}
					Func<CallSite, string, object, object> target6 = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitef.Target;
					CallSite <>p__Sitef = PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Sitef;
					string name = commonUsedShortcut.Name;
					if (PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Site10 == null)
					{
						PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Site10 = CallSite<Func<CallSite, object, object>>.Create(Binder.GetMember(CSharpBinderFlags.None, "Name", typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
					}
					if (target5(<>p__Sitee, target6(<>p__Sitef, name, PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Site10.Target(PreferenceCommonUsedWindow.<DisableMenuItemDelete_Click>o__SiteContainer8.<>p__Site10, selectedItem))))
					{
						this.CommonSection.Remove(commonUsedShortcut);
						this.ProgramName.Text = "";
						this.ProgramDesc.Text = "";
						this.ProgramName.IsEnabled = true;
						this.Add.Visibility = Visibility.Visible;
						this.Modify.Visibility = Visibility.Collapsed;
						this.Cancle_B.Visibility = Visibility.Collapsed;
						break;
					}
				}
			}
		}

		// Token: 0x06000B46 RID: 2886 RVA: 0x000370EC File Offset: 0x000352EC
		private void EnableMenuItemDelete_Click(object sender, RoutedEventArgs e)
		{
			if (this.Enable.SelectedIndex == -1)
			{
				return;
			}
			object selectedItem = this.Enable.SelectedItem;
			if (PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site12 == null)
			{
				PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site12 = CallSite<Func<CallSite, object, bool>>.Create(Binder.UnaryOperation(CSharpBinderFlags.None, ExpressionType.IsTrue, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
			}
			Func<CallSite, object, bool> target = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site12.Target;
			CallSite <>p__Site = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site12;
			if (PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site13 == null)
			{
				PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site13 = CallSite<Func<CallSite, MessageBoxResult, object, object>>.Create(Binder.BinaryOperation(CSharpBinderFlags.None, ExpressionType.Equal, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[]
				{
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType | CSharpArgumentInfoFlags.Constant, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null)
				}));
			}
			Func<CallSite, MessageBoxResult, object, object> target2 = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site13.Target;
			CallSite <>p__Site2 = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site13;
			MessageBoxResult messageBoxResult = MessageBoxResult.Yes;
			if (PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site14 == null)
			{
				PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site14 = CallSite<Func<CallSite, Type, object, string, MessageBoxButton, MessageBoxImage, object>>.Create(Binder.InvokeMember(CSharpBinderFlags.None, "Show", null, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[]
				{
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType | CSharpArgumentInfoFlags.IsStaticType, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType | CSharpArgumentInfoFlags.Constant, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType | CSharpArgumentInfoFlags.Constant, null)
				}));
			}
			Func<CallSite, Type, object, string, MessageBoxButton, MessageBoxImage, object> target3 = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site14.Target;
			CallSite <>p__Site3 = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site14;
			Type typeFromHandle = typeof(DesignerMessageBox);
			if (PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site15 == null)
			{
				PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site15 = CallSite<Func<CallSite, Type, string, object, object>>.Create(Binder.InvokeMember(CSharpBinderFlags.None, "Format", null, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[]
				{
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType | CSharpArgumentInfoFlags.IsStaticType, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType, null),
					CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null)
				}));
			}
			Func<CallSite, Type, string, object, object> target4 = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site15.Target;
			CallSite <>p__Site4 = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site15;
			Type typeFromHandle2 = typeof(string);
			string text = Application.Current.FindResource("Preference_ConfirmDeleteProg") as string;
			if (PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site16 == null)
			{
				PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site16 = CallSite<Func<CallSite, object, object>>.Create(Binder.GetMember(CSharpBinderFlags.None, "Name", typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
			}
			if (target(<>p__Site, target2(<>p__Site2, messageBoxResult, target3(<>p__Site3, typeFromHandle, target4(<>p__Site4, typeFromHandle2, text, PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site16.Target(PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site16, selectedItem)), Application.Current.FindResource("Preference_ConfirmDelete") as string, MessageBoxButton.YesNo, MessageBoxImage.Exclamation))))
			{
				foreach (CommonUsedShortcut commonUsedShortcut in this.CommonSection.Items)
				{
					if (PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site17 == null)
					{
						PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site17 = CallSite<Func<CallSite, object, bool>>.Create(Binder.UnaryOperation(CSharpBinderFlags.None, ExpressionType.IsTrue, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
					}
					Func<CallSite, object, bool> target5 = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site17.Target;
					CallSite <>p__Site5 = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site17;
					if (PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site18 == null)
					{
						PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site18 = CallSite<Func<CallSite, string, object, object>>.Create(Binder.BinaryOperation(CSharpBinderFlags.None, ExpressionType.Equal, typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[]
						{
							CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.UseCompileTimeType, null),
							CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null)
						}));
					}
					Func<CallSite, string, object, object> target6 = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site18.Target;
					CallSite <>p__Site6 = PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site18;
					string name = commonUsedShortcut.Name;
					if (PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site19 == null)
					{
						PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site19 = CallSite<Func<CallSite, object, object>>.Create(Binder.GetMember(CSharpBinderFlags.None, "Name", typeof(PreferenceCommonUsedWindow), new CSharpArgumentInfo[] { CSharpArgumentInfo.Create(CSharpArgumentInfoFlags.None, null) }));
					}
					if (target5(<>p__Site5, target6(<>p__Site6, name, PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site19.Target(PreferenceCommonUsedWindow.<EnableMenuItemDelete_Click>o__SiteContainer11.<>p__Site19, selectedItem))))
					{
						this.CommonSection.Remove(commonUsedShortcut);
						this.ProgramName.Text = "";
						this.ProgramDesc.Text = "";
						this.ProgramName.IsEnabled = true;
						this.Add.Visibility = Visibility.Visible;
						this.Modify.Visibility = Visibility.Collapsed;
						this.Cancle_B.Visibility = Visibility.Collapsed;
						break;
					}
				}
			}
		}

		// Token: 0x06000B47 RID: 2887 RVA: 0x000374D8 File Offset: 0x000356D8
		private void ReloadButton_Click(object sender, RoutedEventArgs e)
		{
			this.CommonSection = new CommonUsedElement();
			base.DataContext = this.CommonSection;
		}

		// Token: 0x06000B48 RID: 2888 RVA: 0x000374F1 File Offset: 0x000356F1
		private void SaveButton_Click(object sender, RoutedEventArgs e)
		{
			PreferenceManager.Current.Save();
		}

		// Token: 0x06000B49 RID: 2889 RVA: 0x00037500 File Offset: 0x00035700
		private void Cancle_Click(object sender, RoutedEventArgs e)
		{
			this.Disable.SelectedIndex = -1;
			this.Enable.SelectedIndex = -1;
			this.ProgramName.IsEnabled = true;
			this.ProgramName.Text = "";
			this.ProgramDesc.Text = "";
			this.Add.Visibility = Visibility.Visible;
			this.Modify.Visibility = Visibility.Collapsed;
			this.Cancle_B.Visibility = Visibility.Collapsed;
		}

		// Token: 0x06000B4A RID: 2890 RVA: 0x0003758C File Offset: 0x0003578C
		public static void ShowProgramNoWindow(TextBox buttonEdit)
		{
			string info_ProgRel = SettingManager.Get().Info_ProgRel;
			string text = (string)Application.Current.FindResource("specProperty_prog_rel");
			MasterDetailView masterDetailView = new MasterDetailView(info_ProgRel, SpecDesigner.Controls.Controls.SelectionMode.Single, buttonEdit.Text);
			Window window = new Window
			{
				Title = text,
				Content = masterDetailView
			};
			masterDetailView.ItemSelected += delegate(object s1, RoutedEventArgs e1)
			{
				window.Close();
			};
			window.ShowDialog();
			if (!string.IsNullOrEmpty(masterDetailView.SelectedID))
			{
				buttonEdit.Text = masterDetailView.SelectedID;
				Keyboard.Focus(buttonEdit);
			}
		}

		// Token: 0x06000B4B RID: 2891 RVA: 0x00037638 File Offset: 0x00035838
		private void s_PreviewMouseLeftButtonDown(object sender, MouseEventArgs e)
		{
			if (sender is ListBoxItem)
			{
				ListBoxItem listBoxItem = sender as ListBoxItem;
				DragDrop.DoDragDrop(listBoxItem, listBoxItem.DataContext, DragDropEffects.Move);
				listBoxItem.IsSelected = true;
			}
		}

		// Token: 0x06000B4C RID: 2892 RVA: 0x0003766C File Offset: 0x0003586C
		private void Disable_DragOver(object sender, DragEventArgs e)
		{
			CommonUsedShortcut commonUsedShortcut = e.Data.GetData(typeof(CommonUsedShortcut)) as CommonUsedShortcut;
			CommonUsedShortcut commonUsedShortcut2 = ((ListBoxItem)sender).DataContext as CommonUsedShortcut;
			int num = this.Disable.Items.IndexOf(commonUsedShortcut);
			int num2 = this.Disable.Items.IndexOf(commonUsedShortcut2);
			if (num < num2)
			{
				this.CommonSection.Move(commonUsedShortcut, num, num2 + 1);
			}
			else
			{
				int num3 = num + 1;
				if (this.CommonSection.DisableList.Count + 1 > num3)
				{
					this.CommonSection.Move(commonUsedShortcut, num3, num2);
				}
			}
			int num4 = 0;
			foreach (CommonUsedShortcut commonUsedShortcut3 in this.CommonSection.DisableList)
			{
				commonUsedShortcut3.Sort = num4++.ToString();
			}
		}

		// Token: 0x06000B4D RID: 2893 RVA: 0x00037768 File Offset: 0x00035968
		private void Enable_DragOver(object sender, DragEventArgs e)
		{
			CommonUsedShortcut commonUsedShortcut = e.Data.GetData(typeof(CommonUsedShortcut)) as CommonUsedShortcut;
			CommonUsedShortcut commonUsedShortcut2 = ((ListBoxItem)sender).DataContext as CommonUsedShortcut;
			int num = this.Enable.Items.IndexOf(commonUsedShortcut);
			int num2 = this.Enable.Items.IndexOf(commonUsedShortcut2);
			if (num < num2)
			{
				this.CommonSection.Move(commonUsedShortcut, num, num2 + 1);
			}
			else
			{
				int num3 = num + 1;
				if (this.CommonSection.EnableList.Count + 1 > num3)
				{
					this.CommonSection.Move(commonUsedShortcut, num3, num2);
				}
			}
			int num4 = 0;
			foreach (CommonUsedShortcut commonUsedShortcut3 in this.CommonSection.EnableList)
			{
				commonUsedShortcut3.Sort = num4++.ToString();
			}
		}

		// Token: 0x06000B4E RID: 2894 RVA: 0x00037864 File Offset: 0x00035A64
		private void ListBox_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			CommonUsedShortcut commonUsedShortcut = (sender as ListBox).SelectedItem as CommonUsedShortcut;
			if (commonUsedShortcut != null)
			{
				this.ProgramName.Text = commonUsedShortcut.Name;
				this.ProgramDesc.Text = commonUsedShortcut.Desc;
				this.Add.Visibility = Visibility.Collapsed;
				this.Modify.Visibility = Visibility.Visible;
				this.Cancle_B.Visibility = Visibility.Visible;
				this.ProgramName.IsEnabled = false;
			}
		}

		// Token: 0x06000B4F RID: 2895 RVA: 0x000378D8 File Offset: 0x00035AD8
		private void Modify_Click(object sender, RoutedEventArgs e)
		{
			foreach (CommonUsedShortcut commonUsedShortcut in this.CommonSection.Items)
			{
				if (commonUsedShortcut.Name == this.ProgramName.Text)
				{
					commonUsedShortcut.Desc = this.ProgramDesc.Text;
					this.CommonSection.Remove(commonUsedShortcut);
					this.CommonSection.Add(commonUsedShortcut);
					this.ProgramName.Text = "";
					this.ProgramDesc.Text = "";
					this.ProgramName.IsEnabled = true;
					this.Add.Visibility = Visibility.Visible;
					this.Modify.Visibility = Visibility.Collapsed;
					this.Cancle_B.Visibility = Visibility.Collapsed;
					break;
				}
			}
		}

		// Token: 0x04000454 RID: 1108
		private CommonUsedElement CommonSection;

		// Token: 0x02000195 RID: 405
		[CompilerGenerated]
		private static class <EnableButton_Click>o__SiteContainer0
		{
			// Token: 0x04000580 RID: 1408
			public static CallSite<Func<CallSite, object, bool>> <>p__Site1;

			// Token: 0x04000581 RID: 1409
			public static CallSite<Func<CallSite, string, object, object>> <>p__Site2;

			// Token: 0x04000582 RID: 1410
			public static CallSite<Func<CallSite, object, object>> <>p__Site3;
		}

		// Token: 0x02000196 RID: 406
		[CompilerGenerated]
		private static class <DisableButton_Click>o__SiteContainer4
		{
			// Token: 0x04000583 RID: 1411
			public static CallSite<Func<CallSite, object, bool>> <>p__Site5;

			// Token: 0x04000584 RID: 1412
			public static CallSite<Func<CallSite, string, object, object>> <>p__Site6;

			// Token: 0x04000585 RID: 1413
			public static CallSite<Func<CallSite, object, object>> <>p__Site7;
		}

		// Token: 0x02000197 RID: 407
		[CompilerGenerated]
		private static class <DisableMenuItemDelete_Click>o__SiteContainer8
		{
			// Token: 0x04000586 RID: 1414
			public static CallSite<Func<CallSite, object, bool>> <>p__Site9;

			// Token: 0x04000587 RID: 1415
			public static CallSite<Func<CallSite, MessageBoxResult, object, object>> <>p__Sitea;

			// Token: 0x04000588 RID: 1416
			public static CallSite<Func<CallSite, Type, object, string, MessageBoxButton, MessageBoxImage, object>> <>p__Siteb;

			// Token: 0x04000589 RID: 1417
			public static CallSite<Func<CallSite, Type, string, object, object>> <>p__Sitec;

			// Token: 0x0400058A RID: 1418
			public static CallSite<Func<CallSite, object, object>> <>p__Sited;

			// Token: 0x0400058B RID: 1419
			public static CallSite<Func<CallSite, object, bool>> <>p__Sitee;

			// Token: 0x0400058C RID: 1420
			public static CallSite<Func<CallSite, string, object, object>> <>p__Sitef;

			// Token: 0x0400058D RID: 1421
			public static CallSite<Func<CallSite, object, object>> <>p__Site10;
		}

		// Token: 0x02000198 RID: 408
		[CompilerGenerated]
		private static class <EnableMenuItemDelete_Click>o__SiteContainer11
		{
			// Token: 0x0400058E RID: 1422
			public static CallSite<Func<CallSite, object, bool>> <>p__Site12;

			// Token: 0x0400058F RID: 1423
			public static CallSite<Func<CallSite, MessageBoxResult, object, object>> <>p__Site13;

			// Token: 0x04000590 RID: 1424
			public static CallSite<Func<CallSite, Type, object, string, MessageBoxButton, MessageBoxImage, object>> <>p__Site14;

			// Token: 0x04000591 RID: 1425
			public static CallSite<Func<CallSite, Type, string, object, object>> <>p__Site15;

			// Token: 0x04000592 RID: 1426
			public static CallSite<Func<CallSite, object, object>> <>p__Site16;

			// Token: 0x04000593 RID: 1427
			public static CallSite<Func<CallSite, object, bool>> <>p__Site17;

			// Token: 0x04000594 RID: 1428
			public static CallSite<Func<CallSite, string, object, object>> <>p__Site18;

			// Token: 0x04000595 RID: 1429
			public static CallSite<Func<CallSite, object, object>> <>p__Site19;
		}
	}
}
